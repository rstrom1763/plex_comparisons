import { createMovieCard } from './gallery.js';
import { FilterPanel } from './filter.js';

let currentRemoteMovies = [];
let currentFilteredMovies = [];
let currentServerId = null;
let filterPanel = null;

const movieProperties = [
    { value: 'title', label: 'Title' },
    { value: 'year', label: 'Year' },
    { value: 'rating', label: 'Content Rating' },
    { value: 'genre', label: 'Genre' },
    { value: 'library', label: 'Library' },
    { value: 'container', label: 'Container' },
    { value: 'video_codec', label: 'Video Codec' },
    { value: 'audio_codec', label: 'Audio Codec' },
    { value: 'resolution', label: 'Resolution' },
    { value: 'bitrate', label: 'Bitrate' },
    { value: 'size', label: 'Size' },
    { value: 'duration', label: 'Duration' },
    { value: 'critic_rating', label: 'Critic Rating' },
    { value: 'audience_rating', label: 'Audience Rating' },
];

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

function sortAndRender(sortConfig) {
    const remoteList = document.getElementById('remote-only-list');
    const countSpan = document.getElementById('item-count');

    if (currentFilteredMovies.length === 0) {
        if (countSpan) countSpan.textContent = '0';
        remoteList.innerHTML = '<p>No movies match the selected filters.</p>';
        return;
    }

    if (countSpan) countSpan.textContent = currentFilteredMovies.length;

    const sortedMovies = [...currentFilteredMovies];
    
    if (sortConfig) {
        sortedMovies.sort((a, b) => {
            let valA = a[sortConfig.property];
            let valB = b[sortConfig.property];

            // Handle strings (titles)
            if (typeof valA === 'string') {
                valA = valA.toLowerCase();
                valB = valB.toLowerCase();
            }

            if (valA < valB) return sortConfig.direction === 'asc' ? -1 : 1;
            if (valA > valB) return sortConfig.direction === 'asc' ? 1 : -1;
            return 0;
        });
    }

    remoteList.innerHTML = '';
    sortedMovies.forEach(m => remoteList.appendChild(createMovieCard(m, currentServerId)));
}

function handleFilterChange(rootGroup, sortConfig) {
    currentFilteredMovies = currentRemoteMovies.filter(movie => rootGroup.apply(movie));
    sortAndRender(sortConfig);
}

async function startComparison() {
    const selector = document.getElementById('server-selector');
    const id = selector.value;
    const name = selector.options[selector.selectedIndex].text;

    if (!id) {
        alert('Please select a server first');
        return;
    }

    // Update URL with server selection
    FilterPanel.patchURL('server', id);

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
        currentFilteredMovies = [...currentRemoteMovies];
        remoteList.innerHTML = '';

        if (currentRemoteMovies.length > 0) {
            sortControls.style.display = 'flex';
            sortAndRender({
                property: filterPanel.sortProperty,
                direction: filterPanel.sortDirection
            });
        } else {
            remoteList.innerHTML = '<p>Nothing unique found on the remote server.</p>';
        }
    } catch (error) {
        console.error(error);
        remoteList.innerHTML = `<p style="color: red">Error: ${error.message}</p>`;
    }
}

function init() {
    const populatePromise = populateServers();
    if (!filterPanel) {
        filterPanel = new FilterPanel('filter-container', movieProperties, handleFilterChange);
    }

    // Check if we should start comparison from URL
    const params = new URLSearchParams(window.location.search);
    const serverId = params.get('server');
    if (serverId) {
        populatePromise.then(() => {
            const selector = document.getElementById('server-selector');
            selector.value = serverId;
            if (selector.value) {
                startComparison();
            }
        });
    }

    window.addEventListener('popstate', () => {
        const params = new URLSearchParams(window.location.search);
        const serverId = params.get('server');
        const selector = document.getElementById('server-selector');
        if (serverId && selector.value !== serverId) {
            selector.value = serverId;
            startComparison();
        } else if (!serverId && selector.value) {
            selector.value = '';
            document.getElementById('comparison-view').style.display = 'none';
        }
    });
}

document.getElementById('compare-btn').addEventListener('click', startComparison);
document.addEventListener('DOMContentLoaded', init);
