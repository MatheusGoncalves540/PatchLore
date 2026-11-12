import { execFileSync } from 'node:child_process';

const API = 'https://api.github.com';

const run = (cmd, args) => {
  try {
    return execFileSync(cmd, args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] }).trim();
  } catch {
    return '';
  }
};

export const githubToken = () => process.env.GITHUB_TOKEN || process.env.GH_TOKEN || run('gh', ['auth', 'token']) || undefined;

// "git@github.com:dono/repo.git" ou "https://github.com/dono/repo" -> "dono/repo"
export function parseRemote(url) {
  return url.match(/github\.com[:/]([^/]+\/[^/]+?)(?:\.git)?\/?$/)?.[1] ?? null;
}

export const detectRepo = () => parseRemote(run('git', ['remote', 'get-url', 'origin']));
export const currentBranch = () => run('git', ['branch', '--show-current']) || null;

export async function findPRForBranch(repo, branch, token) {
  const owner = repo.split('/')[0];
  const [pr] = await gh(`/repos/${repo}/pulls?state=all&head=${owner}:${encodeURIComponent(branch)}`, token);
  return pr?.number ?? null;
}

async function gh(path, token, init = {}) {
  const res = await fetch(API + path, {
    ...init,
    headers: {
      accept: 'application/vnd.github+json',
      'x-github-api-version': '2022-11-28',
      ...(token && { authorization: `Bearer ${token}` }),
      ...init.headers,
    },
  });
  if (!res.ok) throw new Error(`GitHub ${init.method ?? 'GET'} ${path}: ${res.status} ${await res.text()}`);
  return res.status === 204 ? null : res.json();
}

async function all(path, token) {
  const out = [];
  for (let page = 1; ; page++) {
    const batch = await gh(`${path}?per_page=100&page=${page}`, token);
    out.push(...batch);
    if (batch.length < 100) return out;
  }
}

export async function fetchPR(repo, number, token) {
  const base = `/repos/${repo}/pulls/${number}`;
  const [pr, commits, files] = await Promise.all([
    gh(base, token),
    all(`${base}/commits`, token),
    all(`${base}/files`, token),
  ]);
  return {
    repo,
    number: pr.number,
    title: pr.title,
    body: pr.body ?? '',
    url: pr.html_url,
    head: pr.head.ref,
    base: pr.base.ref,
    commits: commits.map((c) => c.commit.message),
    files: files.map((f) => ({ name: f.filename, status: f.status, additions: f.additions, deletions: f.deletions })),
  };
}

// Commits the PNG to a dedicated branch so the PR comment can embed it.
export async function uploadImage(repo, branch, path, pngBuffer, token) {
  try {
    await gh(`/repos/${repo}/git/ref/heads/${branch}`, token);
  } catch {
    const { default_branch } = await gh(`/repos/${repo}`, token);
    const { object } = await gh(`/repos/${repo}/git/ref/heads/${default_branch}`, token);
    await gh(`/repos/${repo}/git/refs`, token, {
      method: 'POST',
      body: JSON.stringify({ ref: `refs/heads/${branch}`, sha: object.sha }),
    });
  }
  let sha;
  try {
    sha = (await gh(`/repos/${repo}/contents/${path}?ref=${branch}`, token)).sha;
  } catch {}
  await gh(`/repos/${repo}/contents/${path}`, token, {
    method: 'PUT',
    body: JSON.stringify({ message: `patchlore: ${path}`, content: pngBuffer.toString('base64'), branch, sha }),
  });
  return `https://github.com/${repo}/blob/${branch}/${path}?raw=true`;
}

const MARKER = '<!-- patchlore -->';

// Creates the PatchLore comment, or edits it in place on later runs.
export async function upsertComment(repo, number, body, token) {
  const comments = await all(`/repos/${repo}/issues/${number}/comments`, token);
  const mine = comments.find((c) => c.body?.startsWith(MARKER));
  const payload = JSON.stringify({ body: `${MARKER}\n${body}` });
  if (mine) return gh(`/repos/${repo}/issues/comments/${mine.id}`, token, { method: 'PATCH', body: payload });
  return gh(`/repos/${repo}/issues/${number}/comments`, token, { method: 'POST', body: payload });
}
