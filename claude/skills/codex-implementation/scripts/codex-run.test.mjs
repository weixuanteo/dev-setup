import assert from "node:assert/strict";
import { chmodSync, mkdirSync, mkdtempSync, readFileSync, realpathSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

const here = fileURLToPath(new URL(".", import.meta.url));
const runner = join(here, "codex-run.mjs");
const fixture = mkdtempSync(join(tmpdir(), "codex-run-test-"));
const fake = join(fixture, "codex");
const runs = join(fixture, "runs");
const cwd = join(fixture, "repo");
mkdirSync(cwd);
writeFileSync(fake, `#!/usr/bin/env node
const fs = require("node:fs");
const args = process.argv.slice(2);
const prompt = fs.readFileSync(0, "utf8");
const outputFlag = args.includes("-o") ? args.indexOf("-o") : args.indexOf("--output-last-message");
const output = args[outputFlag + 1];
const resume = args.indexOf("resume");
const thread = resume >= 0 ? args[resume + 4] : "thread-test";
console.log(JSON.stringify({type:"thread.started", thread_id:thread}));
console.log(JSON.stringify({type:"test.args", args}));
setTimeout(() => {
  fs.writeFileSync(output, (resume >= 0 ? "resumed:" : "done:") + prompt.trim());
  process.exit(prompt.includes("FAIL") ? 7 : 0);
}, prompt.includes("SLOW") ? 800 : 10);
`);
chmodSync(fake, 0o755);

const env = { ...process.env, CODEX_RUNNER_BIN: fake, CODEX_RUNNER_RUNS: runs };

function call(args, ok = true) {
  const result = spawnSync(process.execPath, [runner, ...args], { encoding: "utf8", env });
  if (ok) assert.equal(result.status, 0, result.stderr);
  return result;
}

function prompt(name, content) {
  const path = join(fixture, name);
  writeFileSync(path, content);
  return path;
}

function start(content) {
  return JSON.parse(call(["start", "--cwd", cwd, "--prompt", prompt(`${Date.now()}-${Math.random()}.txt`, content)]).stdout).id;
}

function wait(id) {
  return JSON.parse(call(["wait", id, "--seconds", "5"]).stdout);
}

function argsFor(id) {
  return JSON.parse(readFileSync(join(runs, id, "events.jsonl"), "utf8").trim().split("\n")[1]).args;
}

test("captures completion and resumes the exact thread", () => {
  const first = start("first");
  assert.deepEqual(wait(first), { id: first, state: "succeeded", exitCode: 0, threadId: "thread-test" });
  assert.equal(call(["result", first]).stdout, "done:first");
  assert.equal(argsFor(first)[argsFor(first).indexOf("-C") + 1], realpathSync(cwd));
  assert.equal(argsFor(first)[argsFor(first).indexOf("--sandbox") + 1], "workspace-write");
  const second = JSON.parse(call(["resume", first, "--prompt", prompt("follow-up.txt", "fix tests")]).stdout).id;
  assert.equal(wait(second).state, "succeeded");
  assert.equal(call(["result", second]).stdout, "resumed:fix tests");
  assert.deepEqual(argsFor(second).filter(arg => ["-s", "-C", "--cd"].includes(arg)), []);
});

test("preserves failure exit status", () => {
  const id = start("FAIL");
  assert.equal(wait(id).exitCode, 7);
  assert.notEqual(call(["result", id], false).status, 0);
});

test("locks the worktree while Codex runs", () => {
  const id = start("SLOW");
  const blocked = call(["start", "--cwd", cwd, "--prompt", prompt("blocked.txt", "other")], false);
  assert.match(blocked.stderr, /already owns this worktree/);
  assert.equal(wait(id).state, "succeeded");
});

test("reviews one target without a prompt or resume", () => {
  const id = JSON.parse(call(["review", "--cwd", cwd, "--base", "main"]).stdout).id;
  assert.equal(wait(id).state, "succeeded");
  assert.deepEqual(argsFor(id).filter(arg => !arg.includes(runs)), ["exec", "review", "--json", "-o", "--base", "main"]);
  for (const target of [["--uncommitted"], ["--commit", "HEAD"]]) {
    const run = JSON.parse(call(["review", "--cwd", cwd, ...target]).stdout).id;
    assert.equal(wait(run).state, "succeeded");
    assert.deepEqual(argsFor(run).slice(-target.length), target);
  }
  assert.match(call(["resume", id, "--prompt", prompt("review-follow-up.txt", "again")], false).stderr, /must start fresh/);
  assert.match(call(["review", "--cwd", cwd, "--base", "main", "--commit", "HEAD"], false).stderr, /exactly one/);
  assert.match(call(["review", "--cwd", cwd, "--base", "main", "--prompt", "extra"], false).stderr, /accepts only/);
});
