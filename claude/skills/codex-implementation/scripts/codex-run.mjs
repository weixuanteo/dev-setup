#!/usr/bin/env node

import { createHash, randomBytes } from "node:crypto";
import { closeSync, existsSync, mkdirSync, openSync, readFileSync, realpathSync, renameSync, unlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const self = fileURLToPath(import.meta.url);
const root = process.env.CODEX_RUNNER_RUNS || join(tmpdir(), `codex-runner-${process.getuid?.() ?? "user"}`);

function die(message) {
  throw new Error(message);
}

function parse(args) {
  const positionals = [];
  const options = {};
  const flags = new Set(["--uncommitted"]);
  for (let i = 0; i < args.length; i++) {
    const value = args[i];
    if (!value.startsWith("--")) positionals.push(value);
    else if (flags.has(value)) options[value.slice(2)] = true;
    else {
      const next = args[++i];
      if (!next || next.startsWith("--")) die(`missing value for ${value}`);
      options[value.slice(2)] = next;
    }
  }
  return { positionals, options };
}

function runDir(id) {
  if (!/^[a-z0-9-]+$/.test(id)) die("invalid run id");
  const dir = join(root, id);
  if (!existsSync(dir)) die(`unknown run: ${id}`);
  return dir;
}

function readJSON(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

function atomicWrite(path, value) {
  const temporary = `${path}.${process.pid}.tmp`;
  writeFileSync(temporary, value);
  renameSync(temporary, path);
}

function alive(pid) {
  try {
    process.kill(pid, 0);
    return true;
  } catch (error) {
    return error.code === "EPERM";
  }
}

function lockPath(cwd) {
  return join(root, `${createHash("sha256").update(cwd).digest("hex").slice(0, 20)}.lock`);
}

function acquireLock(path, id) {
  mkdirSync(root, { recursive: true });
  for (;;) {
    try {
      const fd = openSync(path, "wx");
      writeFileSync(fd, JSON.stringify({ id, pid: process.pid }));
      closeSync(fd);
      return;
    } catch (error) {
      if (error.code !== "EEXIST") throw error;
      let lock;
      try { lock = readJSON(path); } catch {}
      if (lock?.pid && alive(lock.pid)) die(`active run ${lock.id} already owns this worktree`);
      unlinkSync(path);
    }
  }
}

function updateLock(path, id) {
  const lock = readJSON(path);
  if (lock.id === id) atomicWrite(path, JSON.stringify({ id, pid: process.pid }));
}

function setLockPid(path, id, pid) {
  try {
    if (readJSON(path).id === id) atomicWrite(path, JSON.stringify({ id, pid }));
  } catch {}
}

function releaseLock(path, id) {
  try {
    if (readJSON(path).id === id) unlinkSync(path);
  } catch {}
}

function threadId(dir) {
  const path = join(dir, "events.jsonl");
  if (!existsSync(path)) return null;
  for (const line of readFileSync(path, "utf8").split("\n")) {
    try {
      const event = JSON.parse(line);
      if (event.type === "thread.started") return event.thread_id;
    } catch {}
  }
  return null;
}

function status(id) {
  const dir = runDir(id);
  const meta = readJSON(join(dir, "meta.json"));
  const exitPath = join(dir, "exit-code");
  const thread = threadId(dir);
  if (existsSync(exitPath)) {
    const exitCode = Number(readFileSync(exitPath, "utf8"));
    const complete = exitCode === 0 && existsSync(join(dir, "final.txt"));
    return { id, state: complete ? "succeeded" : "failed", exitCode, threadId: thread };
  }
  let lock;
  try { lock = readJSON(meta.lock); } catch {}
  return { id, state: lock?.id === id && alive(lock.pid) ? "running" : "failed", exitCode: null, threadId: thread };
}

function startRun({ cwd, promptPath, review, thread }) {
  cwd = realpathSync(cwd);
  const prompt = promptPath ? readFileSync(promptPath, "utf8") : null;
  if (prompt !== null && !prompt.trim()) die("prompt is empty");
  if (prompt === null && !review) die("run requires a prompt or review target");
  const id = `${Date.now().toString(36)}-${randomBytes(4).toString("hex")}`;
  const dir = join(root, id);
  const lock = lockPath(cwd);
  acquireLock(lock, id);
  mkdirSync(dir, { recursive: true });
  if (prompt !== null) writeFileSync(join(dir, "prompt.txt"), prompt);
  writeFileSync(join(dir, "meta.json"), JSON.stringify({ id, cwd, review, thread, lock }));
  try {
    const child = spawn(process.execPath, [self, "_run", id], { detached: true, stdio: "ignore" });
    setLockPid(lock, id, child.pid);
    child.unref();
  } catch (error) {
    releaseLock(lock, id);
    throw error;
  }
  return { id, state: "running" };
}

async function supervise(id) {
  const dir = runDir(id);
  const meta = readJSON(join(dir, "meta.json"));
  updateLock(meta.lock, id);
  const prompt = meta.review ? "ignore" : openSync(join(dir, "prompt.txt"), "r");
  const events = openSync(join(dir, "events.jsonl"), "a");
  const errors = openSync(join(dir, "stderr.log"), "a");
  const codex = process.env.CODEX_RUNNER_BIN || "codex";
  const output = join(dir, "final.txt");
  const args = meta.review
    ? ["exec", "review", "--json", "-o", output, ...meta.review]
    : meta.thread
    ? ["exec", "resume", "--json", "-o", output, meta.thread, "-"]
    : ["exec", "--json", "-C", meta.cwd, "--sandbox", "workspace-write", "-o", output, "-"];
  let exitCode = 1;
  try {
    const child = spawn(codex, args, { cwd: meta.cwd, stdio: [prompt, events, errors] });
    exitCode = await new Promise((resolve, reject) => {
      child.once("error", reject);
      child.once("close", (code, signal) => resolve(code ?? (signal ? 128 : 1)));
    });
  } catch (error) {
    writeFileSync(errors, `${error.stack || error}\n`);
  } finally {
    if (typeof prompt === "number") closeSync(prompt);
    closeSync(events);
    closeSync(errors);
    atomicWrite(join(dir, "exit-code"), String(exitCode));
    releaseLock(meta.lock, id);
  }
}

async function main() {
  mkdirSync(root, { recursive: true });
  const [command, ...rest] = process.argv.slice(2);
  if (command === "_run") return supervise(rest[0]);
  const { positionals, options } = parse(rest);
  if (command === "start") {
    if (!options.cwd || !options.prompt) die("start requires --cwd and --prompt");
    console.log(JSON.stringify(startRun({ cwd: options.cwd, promptPath: options.prompt })));
    return;
  }
  if (command === "review") {
    if (!options.cwd) die("review requires --cwd");
    const unknown = Object.keys(options).filter(option => !["cwd", "uncommitted", "base", "commit"].includes(option));
    if (positionals.length || unknown.length) die("review accepts only a working directory and one target");
    const targets = [
      options.uncommitted && ["--uncommitted"],
      options.base && ["--base", options.base],
      options.commit && ["--commit", options.commit],
    ].filter(Boolean);
    if (targets.length !== 1) die("review requires exactly one of --uncommitted, --base, or --commit");
    console.log(JSON.stringify(startRun({ cwd: options.cwd, review: targets[0] })));
    return;
  }
  if (command === "resume") {
    const previous = status(positionals[0]);
    if (previous.state === "running") die("cannot resume a running run");
    if (!previous.threadId) die("run has no Codex thread id");
    if (!options.prompt) die("resume requires --prompt");
    const meta = readJSON(join(runDir(previous.id), "meta.json"));
    if (meta.review) die("reviews must start fresh");
    console.log(JSON.stringify(startRun({ cwd: meta.cwd, promptPath: options.prompt, thread: previous.threadId })));
    return;
  }
  if (command === "status") {
    console.log(JSON.stringify(status(positionals[0])));
    return;
  }
  if (command === "wait") {
    const deadline = Date.now() + Math.min(Number(options.seconds || 45), 55) * 1000;
    let current;
    do {
      current = status(positionals[0]);
      if (current.state !== "running") break;
      await new Promise(resolve => setTimeout(resolve, 500));
    } while (Date.now() < deadline);
    console.log(JSON.stringify(current));
    return;
  }
  if (command === "result") {
    const current = status(positionals[0]);
    const dir = runDir(current.id);
    if (current.state === "running") die("run is still running");
    if (current.state === "succeeded") process.stdout.write(readFileSync(join(dir, "final.txt"), "utf8"));
    else {
      const stderr = existsSync(join(dir, "stderr.log")) ? readFileSync(join(dir, "stderr.log"), "utf8") : "";
      const events = existsSync(join(dir, "events.jsonl")) ? readFileSync(join(dir, "events.jsonl"), "utf8") : "";
      process.stderr.write((stderr || events).slice(-8000) || `Codex exited ${current.exitCode ?? "without status"}\n`);
      process.exitCode = 1;
    }
    return;
  }
  die("usage: codex-run.mjs start|review|resume|status|wait|result");
}

main().catch(error => {
  process.stderr.write(`error: ${error.message}\n`);
  process.exitCode = 1;
});
