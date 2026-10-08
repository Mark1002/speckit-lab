'use strict';

const app = document.getElementById('app');
const bar = document.getElementById('bar');
const dialog = document.getElementById('task-dialog');
const ROLES = { product_manager: 'Product manager', engineer: 'Engineer' };
const COLUMNS = [
  ['todo', 'To Do'],
  ['in_progress', 'In Progress'],
  ['in_review', 'In Review'],
  ['done', 'Done'],
];
let users = [];

// Strings passed as children become text nodes, never markup (FR-026).
function el(tag, attrs = {}, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (key.startsWith('on')) node.addEventListener(key.slice(2), value);
    else node.setAttribute(key, value);
  }
  node.append(...children);
  return node;
}

function currentUser() {
  try {
    return Number(localStorage.getItem('taskifyUser')) || 0;
  } catch {
    return 0;
  }
}

function setUser(id) {
  try {
    if (id) localStorage.setItem('taskifyUser', String(id));
    else localStorage.removeItem('taskifyUser');
  } catch {
    // Storage blocked: the user is asked to pick again on the next page.
  }
}

async function api(method, path, body) {
  const options = { method, headers: { 'X-Taskify-User': String(currentUser()) } };
  if (body !== undefined) {
    options.headers['Content-Type'] = 'application/json';
    options.body = JSON.stringify(body);
  }
  let res;
  try {
    res = await fetch(path, options);
  } catch {
    return { ok: false, error: { message: 'Taskify is not reachable. Is it still running?' } };
  }
  const data = res.status === 204 ? null : await res.json().catch(() => null);
  if (res.status === 401) {
    setUser(0);
    location.hash = '#/';
  }
  if (res.ok) return { ok: true, data };
  return { ok: false, error: (data && data.error) || { message: 'Something went wrong.' } };
}

function showError(target, error) {
  target.replaceChildren(el('p', { class: 'error', role: 'alert' }, error.message));
}

function renderBar() {
  const me = users.find((u) => u.id === currentUser());
  bar.replaceChildren(
    el('a', { class: 'brand', href: me ? '#/projects' : '#/' }, 'Taskify'),
    ...(me
      ? [
          el('span', { class: 'who' }, `Working as ${me.name}`),
          el('button', { type: 'button', onclick: () => { location.hash = '#/'; } }, 'Switch user'),
        ]
      : []),
  );
}

function showPicker() {
  app.replaceChildren(
    el('h1', {}, 'Who are you?'),
    el('ul', { class: 'users' }, ...users.map((u) => el('li', {},
      el('button', {
        type: 'button',
        onclick: () => {
          setUser(u.id);
          location.hash = '#/projects';
        },
      }, el('span', { class: 'name' }, u.name), el('span', { class: 'role' }, ROLES[u.role] || u.role)),
    ))),
  );
}

async function showProjects() {
  const r = await api('GET', '/api/projects');
  if (!r.ok) return showError(app, r.error);
  app.replaceChildren(
    el('h1', {}, 'Projects'),
    el('ul', { class: 'projects' }, ...r.data.map((p) => el('li', {}, el('a', { href: `#/projects/${p.id}` }, p.name)))),
  );
}

async function moveTask(pid, tid, status, errorBox) {
  const r = await api('PATCH', `/api/projects/${pid}/tasks/${tid}`, { status });
  if (!r.ok) {
    showError(errorBox, r.error);
    return false;
  }
  await showBoard(pid);
  return true;
}

function openTask(pid, task) {
  const errorBox = el('div');
  const column = el('select', { id: 'task-column' }, ...COLUMNS.map(([value, label]) => el('option', { value }, label)));
  column.value = task.status;
  column.addEventListener('change', async () => {
    if (await moveTask(pid, task.id, column.value, errorBox)) {
      task.status = column.value;
      errorBox.replaceChildren();
    } else {
      column.value = task.status;
    }
  });
  dialog.replaceChildren(
    el('h2', {}, task.title),
    el('label', { for: 'task-column' }, 'Column'),
    column,
    errorBox,
    el('button', { type: 'button', onclick: () => dialog.close() }, 'Close'),
  );
  dialog.showModal();
}

function card(pid, task) {
  const assignee = users.find((u) => u.id === task.assigneeId);
  return el('button', {
    type: 'button',
    class: task.assigneeId === currentUser() ? 'card mine' : 'card',
    draggable: 'true',
    onclick: () => openTask(pid, task),
    ondragstart: (e) => e.dataTransfer.setData('text/plain', String(task.id)),
  }, el('span', { class: 'title' }, task.title), el('span', { class: 'assignee' }, assignee ? assignee.name : 'Unassigned'));
}

async function showBoard(pid) {
  const r = await api('GET', `/api/projects/${pid}`);
  if (!r.ok) return showError(app, r.error);
  const project = r.data;
  const errorBox = el('div');
  const columns = COLUMNS.map(([status, label]) => {
    const column = el('section', { class: 'column' },
      el('h2', {}, label),
      ...project.tasks.filter((t) => t.status === status).map((t) => card(pid, t)));
    column.addEventListener('dragover', (e) => {
      e.preventDefault();
      column.classList.add('over');
    });
    column.addEventListener('dragleave', (e) => {
      if (!column.contains(e.relatedTarget)) column.classList.remove('over');
    });
    column.addEventListener('drop', (e) => {
      e.preventDefault();
      column.classList.remove('over');
      const task = project.tasks.find((t) => t.id === Number(e.dataTransfer.getData('text/plain')));
      if (task && task.status !== status) moveTask(pid, task.id, status, errorBox);
    });
    return column;
  });
  app.replaceChildren(
    el('p', {}, el('a', { href: '#/projects' }, '← All projects')),
    el('h1', {}, project.name),
    errorBox,
    el('div', { class: 'board' }, ...columns),
  );
}

async function route() {
  if (!users.length) {
    const r = await api('GET', '/api/users');
    if (!r.ok) return showError(app, r.error);
    users = r.data;
  }
  const hash = location.hash || '#/';
  if (hash !== '#/' && !currentUser()) {
    location.hash = '#/';
    return;
  }
  renderBar();
  const board = hash.match(/^#\/projects\/(\d+)$/);
  if (hash === '#/') return showPicker();
  if (hash === '#/projects') return showProjects();
  if (board) return showBoard(Number(board[1]));
  location.hash = '#/projects';
}

window.addEventListener('hashchange', route);
route();
