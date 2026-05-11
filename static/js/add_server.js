let servers = [];

async function fetchServers() {
    const response = await fetch('/api/servers');
    servers = await response.json();
    const list = document.getElementById('server-list');
    list.innerHTML = '';
    
    servers.forEach(server => {
        const li = document.createElement('li');
        li.className = 'server-item';
        li.innerHTML = `
            <span><strong>${server.name}</strong> (${server.address})</span>
            <div style="display: flex; gap: 10px;">
                <button class="btn btn-primary btn-edit" data-id="${server.id}">Edit</button>
                <button class="btn btn-danger btn-delete" data-id="${server.id}">Delete</button>
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
}

function startEdit(id) {
    const server = servers.find(s => s.id == id);
    if (!server) return;

    document.getElementById('form-title').textContent = 'Edit Server';
    document.getElementById('server-id').value = server.id;
    document.getElementById('server-name').value = server.name;
    document.getElementById('server-address').value = server.address;
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

function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

async function handleFormSubmit(e) {
    e.preventDefault();
    const id = document.getElementById('server-id').value;
    const name = document.getElementById('server-name').value;
    const address = document.getElementById('server-address').value;

    const method = id ? 'PUT' : 'POST';
    const url = id ? `/api/servers/${id}` : '/api/servers';

    const response = await fetch(url, {
        method: method,
        headers: { 
            'Content-Type': 'application/json',
            'X-CSRF-Token': getCookie('csrf_token')
        },
        body: JSON.stringify({ name, address })
    });

    if (response.ok) {
        cancelEdit();
        fetchServers();
    } else {
        alert(`Failed to ${id ? 'update' : 'add'} server`);
    }
}

async function deleteServer(id) {
    if (!confirm('Are you sure?')) return;
    const response = await fetch(`/api/servers/${id}`, { 
        method: 'DELETE',
        headers: {
            'X-CSRF-Token': getCookie('csrf_token')
        }
    });
    if (response.ok) {
        if (document.getElementById('server-id').value == id) {
            cancelEdit();
        }
        fetchServers();
    }
}

document.getElementById('add-server-form').addEventListener('submit', handleFormSubmit);
document.getElementById('cancel-edit').addEventListener('click', cancelEdit);
document.addEventListener('DOMContentLoaded', fetchServers);
