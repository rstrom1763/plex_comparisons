import { createMovieCard } from './gallery.js';

let currentRemoteMovies = [];
let currentServerId = null;

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

function sortAndRender() {
    const property = document.getElementById('sort-property').value;
    const direction = document.getElementById('sort-direction').value;
    const remoteList = document.getElementById('remote-only-list');

    if (currentRemoteMovies.length === 0) return;

    const sortedMovies = [...currentRemoteMovies].sort((a, b) => {
        let valA = a[property];
        let valB = b[property];

        // Handle strings (titles)
        if (typeof valA === 'string') {
            valA = valA.toLowerCase();
            valB = valB.toLowerCase();
        }

        if (valA < valB) return direction === 'asc' ? -1 : 1;
        if (valA > valB) return direction === 'asc' ? 1 : -1;
        return 0;
    });

    remoteList.innerHTML = '';
    sortedMovies.forEach(m => remoteList.appendChild(createMovieCard(m, currentServerId)));
}

async function startComparison() {
    const selector = document.getElementById('server-selector');
    const id = selector.value;
    const name = selector.options[selector.selectedIndex].text;

    if (!id) {
        alert('Please select a server first');
        return;
    }

    currentServerId = id;
    const view = document.getElementById('comparison-view');
    const title = document.getElementById('comparing-with-title');
    const remoteList = document.getElementById('remote-only-list');
    const sortControls = document.getElementById('sort-controls');

    view.style.display = 'block';
    title.innerText = `Comparing with ${name}...`;
    remoteList.innerHTML = 'Loading...';
    sortControls.style.display = 'none';

    try {
        const response = await fetch(`/api/compare/${id}`);
        if (!response.ok) throw new Error(await response.text());
        const data = await response.json();

        currentRemoteMovies = data.remote_only || [];
        remoteList.innerHTML = '';

        if (currentRemoteMovies.length > 0) {
            sortControls.style.display = 'flex';
            sortAndRender();
        } else {
            remoteList.innerHTML = '<p>Nothing unique found on the remote server.</p>';
        }
    } catch (error) {
        console.error(error);
        remoteList.innerHTML = `<p style="color: red">Error: ${error.message}</p>`;
    }
}

document.getElementById('compare-btn').addEventListener('click', startComparison);
document.getElementById('sort-property').addEventListener('change', sortAndRender);
document.getElementById('sort-direction').addEventListener('change', sortAndRender);
document.addEventListener('DOMContentLoaded', populateServers);
