import { authenticatedFetch, handleFetchError } from '/static/js/utils.js';

let servers = [];

async function fetchServers() {
    const [serverResponse, localSnapshotResponse] = await Promise.all([
        fetch('/api/servers'),
        fetch('/api/snapshots/local')
    ]);
    if (!serverResponse.ok) return await handleFetchError(serverResponse, 'fetch servers');
    servers = await serverResponse.json();
    const localSnapshot = localSnapshotResponse.ok ? await localSnapshotResponse.json() : { snapshot_age: 'unknown' };

    const listedServers = [
        {
            id: 'local',
            name: 'local',
            address: window.location.origin,
            snapshot_age: localSnapshot.snapshot_age || 'never',
            local: true
        },
        ...servers
    ];

    const list = document.getElementById('server-list');
    list.innerHTML = '';

    listedServers.forEach(server => {
        const li = document.createElement('li');
        li.className = 'server-item';
        li.innerHTML = `
            <div style="flex-grow: 1;">
                <strong>${server.name}</strong> (${server.address})
                ${server.local ? '<br><small style="color: var(--accent-color);">Local server</small>' : (server.token ? '<br><small style="color: var(--accent-color);">Token configured</small>' : '<br><small style="color: #ff4444;">No token configured</small>')}
                <br><small>Snapshot: ${server.snapshot_age || 'never'}</small>
            </div>
            <div style="display: flex; gap: 10px; align-items: center;">
                <button class="btn btn-primary btn-snapshot" data-id="${server.id}">Take snapshot</button>
                ${server.local ? '' : `<button class="btn btn-primary btn-edit" data-id="${server.id}">Edit</button>
                <button class="btn btn-danger btn-delete" data-id="${server.id}">Delete</button>`}
            </div>
        `;
        list.appendChild(li);
    });

    // Add event listeners
    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', () => deleteServer(btn.dataset.id));
    });
    document.querySelectorAll('.btn-edit').forEach(btn => {
        btn.addEventListener('click', () => startEdit(btn.dataset.id));
    });
    document.querySelectorAll('.btn-snapshot').forEach(btn => {
        btn.addEventListener('click', () => takeSnapshot(btn.dataset.id));
    });
}

function startEdit(id) {
    const server = servers.find(s => s.id == id);
    if (!server) return;

    document.getElementById('form-title').textContent = 'Edit Server';
    document.getElementById('server-id').value = server.id;
    document.getElementById('server-name').value = server.name;
    document.getElementById('server-address').value = server.address;
    document.getElementById('server-token').value = server.token || '';
    document.getElementById('submit-btn').textContent = 'Update Server';
    document.getElementById('cancel-edit').style.display = 'inline-block';
}

function cancelEdit() {
    document.getElementById('form-title').textContent = 'Add Server';
    document.getElementById('server-id').value = '';
    document.getElementById('add-server-form').reset();
    document.getElementById('submit-btn').textContent = 'Add Server';
    document.getElementById('cancel-edit').style.display = 'none';
}

async function handleFormSubmit(e) {
    e.preventDefault();
    const id = document.getElementById('server-id').value;
    const name = document.getElementById('server-name').value;
    const address = document.getElementById('server-address').value;
    const token = document.getElementById('server-token').value;

    const method = id ? 'PUT' : 'POST';
    const url = id ? `/api/servers/${id}` : '/api/servers';

    const response = await authenticatedFetch(url, {
        method: method,
        headers: { 
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ name, address, token })
    });

    if (response.ok) {
        cancelEdit();
        fetchServers();
    } else {
        await handleFetchError(response, id ? 'update server' : 'add server');
    }
}

async function deleteServer(id) {
    if (!confirm('Are you sure?')) return;
    const response = await authenticatedFetch(`/api/servers/${id}`, { 
        method: 'DELETE'
    });
    if (response.ok) {
        if (document.getElementById('server-id').value == id) {
            cancelEdit();
        }
        fetchServers();
    } else {
        await handleFetchError(response, 'delete server');
    }
}

async function takeSnapshot(id) {
    const button = document.querySelector(`.btn-snapshot[data-id="${id}"]`);
    if (button) {
        button.disabled = true;
        button.textContent = 'Taking snapshot...';
    }

    const url = id === 'local' ? '/api/snapshots/local' : `/api/servers/${id}/snapshot`;
    const response = await authenticatedFetch(url, { method: 'POST' });
    if (response.ok) {
        fetchServers();
    } else {
        await handleFetchError(response, 'take snapshot');
        fetchServers();
    }
}

document.getElementById('add-server-form').addEventListener('submit', handleFormSubmit);
document.getElementById('cancel-edit').addEventListener('click', cancelEdit);
document.addEventListener('DOMContentLoaded', fetchServers);
