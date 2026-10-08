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
  const name = el('input');
  const description = el('textarea', { rows: '3' });
  const form = el('form', {
    class: 'new-project',
    onsubmit: async (e) => {
      e.preventDefault();
      const res = await api('POST', '/api/projects', { name: name.value, description: description.value });
      if (!res.ok) return showFieldError(form, res.error);
      showProjects();
    },
  }, el('h2', {}, 'New project'), field('Name', name, 'name'), field('Description', description, 'description'),
  formError(), el('button', { type: 'submit' }, 'Create project'));
  app.replaceChildren(
    el('h1', {}, 'Projects'),
    el('ul', { class: 'projects' }, ...r.data.map((p) => el('li', {}, el('a', { href: `#/projects/${p.id}` }, p.name)))),
    form,
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

function userName(id) {
  return users.find((u) => u.id === id)?.name ?? 'Unassigned';
}

let fieldCount = 0;

function field(label, control, name) {
  control.id = `field-${++fieldCount}`;
  return el('div', { class: 'field' },
    el('label', { for: control.id }, label),
    control,
    el('p', { class: 'field-error', 'data-field': name }));
}

function formError() {
  return el('p', { class: 'field-error', 'data-field': '', role: 'alert' });
}

function showFieldError(form, error) {
  const slots = [...form.querySelectorAll('.field-error')];
  for (const slot of slots) slot.textContent = '';
  const slot = slots.find((s) => s.dataset.field === error.field) || slots.find((s) => s.dataset.field === '');
  slot.textContent = error.message;
}

function assigneeSelect(value) {
  const select = el('select', {},
    el('option', { value: '0' }, 'Unassigned'),
    ...users.map((u) => el('option', { value: String(u.id) }, u.name)));
  select.value = String(value);
  return select;
}

function closeButton() {
  return el('button', { type: 'button', onclick: () => dialog.close() }, 'Close');
}

// showModal throws if the dialog is already open, which it is when content is reloaded.
function openDialog(...children) {
  dialog.replaceChildren(...children);
  if (!dialog.open) dialog.showModal();
}

function openNewTask(pid) {
  const title = el('input');
  const description = el('textarea', { rows: '4' });
  const assignee = assigneeSelect(0);
  const form = el('form', {
    onsubmit: async (e) => {
      e.preventDefault();
      const r = await api('POST', `/api/projects/${pid}/tasks`, {
        title: title.value, description: description.value, assigneeId: Number(assignee.value),
      });
      if (!r.ok) return showFieldError(form, r.error);
      dialog.close();
      showBoard(pid);
    },
  }, field('Title', title, 'title'), field('Description', description, 'description'),
  field('Assignee', assignee, 'assigneeId'), formError(), el('button', { type: 'submit' }, 'Create task'));
  openDialog(el('h2', {}, 'New task'), form, closeButton());
}

async function openTask(pid, tid) {
  const r = await api('GET', `/api/projects/${pid}/tasks/${tid}`);
  if (!r.ok) return openDialog(el('p', { class: 'error', role: 'alert' }, r.error.message), closeButton());
  const task = r.data;
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

  const title = el('input', { value: task.title });
  const description = el('textarea', { rows: '4' }, task.description);
  const assignee = assigneeSelect(task.assigneeId);
  const form = el('form', {
    onsubmit: async (e) => {
      e.preventDefault();
      const changes = {};
      if (title.value !== task.title) changes.title = title.value;
      if (description.value !== task.description) changes.description = description.value;
      if (Number(assignee.value) !== task.assigneeId) changes.assigneeId = Number(assignee.value);
      if (!Object.keys(changes).length) return;
      const res = await api('PATCH', `/api/projects/${pid}/tasks/${tid}`, changes);
      if (!res.ok) return showFieldError(form, res.error);
      await showBoard(pid);
      openTask(pid, tid);
    },
  }, field('Title', title, 'title'), field('Description', description, 'description'),
  field('Assignee', assignee, 'assigneeId'), formError(), el('button', { type: 'submit' }, 'Save'));

  openDialog(
    el('h2', {}, task.title),
    el('p', { class: 'description' }, task.description || 'No description.'),
    el('p', { class: 'meta' }, `Assigned to ${userName(task.assigneeId)} · Created by ${userName(task.createdBy)}`),
    el('label', { for: 'task-column' }, 'Column'),
    column,
    errorBox,
    el('details', {}, el('summary', {}, 'Edit task'), form),
    el('section', { class: 'comments' },
      el('h3', {}, 'Comments'),
      task.comments.length
        ? el('ul', { class: 'comment-list' }, ...task.comments.map((c) => commentItem(pid, tid, c)))
        : el('p', { class: 'meta' }, 'No comments yet.'),
      newCommentForm(pid, tid)),
    closeButton(),
  );
}

function commentItem(pid, tid, comment) {
  const path = `/api/projects/${pid}/tasks/${tid}/comments/${comment.id}`;
  const item = el('li', { class: 'comment' });
  const edit = () => {
    const text = el('textarea', { rows: '3' }, comment.text);
    const form = el('form', {
      onsubmit: async (e) => {
        e.preventDefault();
        const r = await api('PATCH', path, { text: text.value });
        if (!r.ok) return showFieldError(form, r.error);
        openTask(pid, tid);
      },
    }, field('Edit comment', text, 'text'), formError(),
    el('button', { type: 'submit' }, 'Save'),
    el('button', { type: 'button', onclick: show }, 'Cancel'));
    item.replaceChildren(form);
  };
  const remove = async () => {
    if (!confirm('Delete this comment? This cannot be undone.')) return;
    const r = await api('DELETE', path);
    if (!r.ok) return showFieldError(item, r.error);
    openTask(pid, tid);
  };
  function show() {
    item.replaceChildren(
      el('p', { class: 'meta' }, `${userName(comment.authorId)} · ${new Date(comment.createdAt).toLocaleString()}`),
      el('p', { class: 'text' }, comment.text),
    );
    if (comment.authorId === currentUser()) {
      item.append(
        el('div', { class: 'actions' },
          el('button', { type: 'button', onclick: edit }, 'Edit'),
          el('button', { type: 'button', onclick: remove }, 'Delete')),
        formError(),
      );
    }
  }
  show();
  return item;
}

function newCommentForm(pid, tid) {
  const text = el('textarea', { rows: '3' });
  const form = el('form', {
    onsubmit: async (e) => {
      e.preventDefault();
      const r = await api('POST', `/api/projects/${pid}/tasks/${tid}/comments`, { text: text.value });
      if (!r.ok) return showFieldError(form, r.error);
      openTask(pid, tid);
    },
  }, field('Add comment', text, 'text'), formError(), el('button', { type: 'submit' }, 'Post'));
  return form;
}

function card(pid, task) {
  return el('button', {
    type: 'button',
    class: task.assigneeId === currentUser() ? 'card mine' : 'card',
    draggable: 'true',
    onclick: () => openTask(pid, task.id),
    ondragstart: (e) => e.dataTransfer.setData('text/plain', String(task.id)),
  }, el('span', { class: 'title' }, task.title), el('span', { class: 'assignee' }, userName(task.assigneeId)));
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
    el('button', { type: 'button', onclick: () => openNewTask(pid) }, 'New task'),
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
