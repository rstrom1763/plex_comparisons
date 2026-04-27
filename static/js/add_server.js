async function fetchServers() {
    const response = await fetch('/api/servers');
    const servers = await response.json();
    const list = document.getElementById('server-list');
    list.innerHTML = '';
    
    servers.forEach(server => {
        const li = document.createElement('li');
        li.className = 'server-item';
        li.innerHTML = `
            <span><strong>${server.name}</strong> (${server.address})</span>
            <div>
                <button class="btn btn-danger btn-delete" data-id="${server.id}">Delete</button>
            </div>
        `;
        list.appendChild(li);
    });

    // Add event listeners
    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', () => deleteServer(btn.dataset.id));
    });
}

async function addServer(e) {
    e.preventDefault();
    const name = document.getElementById('server-name').value;
    const address = document.getElementById('server-address').value;

    const response = await fetch('/api/servers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, address })
    });

    if (response.ok) {
        document.getElementById('add-server-form').reset();
        fetchServers();
    } else {
        alert('Failed to add server');
    }
}

async function deleteServer(id) {
    if (!confirm('Are you sure?')) return;
    const response = await fetch(`/api/servers/${id}`, { method: 'DELETE' });
    if (response.ok) fetchServers();
}

document.getElementById('add-server-form').addEventListener('submit', addServer);
document.addEventListener('DOMContentLoaded', fetchServers);
