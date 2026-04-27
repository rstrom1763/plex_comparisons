import { createMovieCard } from './gallery.js';

async function populateServers() {
    try {
        const response = await fetch('/api/servers');
        const servers = await response.json();
        const selector = document.getElementById('server-selector');
        
        servers.forEach(server => {
            const option = document.createElement('option');
            option.value = server.id;
            option.textContent = server.name;
            selector.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to fetch servers:', error);
    }
}

async function startComparison() {
    const selector = document.getElementById('server-selector');
    const id = selector.value;
    const name = selector.options[selector.selectedIndex].text;

    if (!id) {
        alert('Please select a server first');
        return;
    }

    const view = document.getElementById('comparison-view');
    const title = document.getElementById('comparing-with-title');
    const remoteList = document.getElementById('remote-only-list');

    view.style.display = 'block';
    title.innerText = `Comparing with ${name}...`;
    remoteList.innerHTML = 'Loading...';

    try {
        const response = await fetch(`/api/compare/${id}`);
        if (!response.ok) throw new Error(await response.text());
        const data = await response.json();

        remoteList.innerHTML = '';

        if (data.remote_only && data.remote_only.length > 0) {
            data.remote_only.forEach(m => remoteList.appendChild(createMovieCard(m)));
        } else {
            remoteList.innerHTML = '<p>Nothing unique found on the remote server.</p>';
        }
    } catch (error) {
        console.error(error);
        remoteList.innerHTML = `<p style="color: red">Error: ${error.message}</p>`;
    }
}

document.getElementById('compare-btn').addEventListener('click', startComparison);
document.addEventListener('DOMContentLoaded', populateServers);
