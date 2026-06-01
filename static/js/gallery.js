import { formatSize, formatDuration, getCookie, authenticatedFetch } from '/static/js/utils.js';

/**
 * Creates a movie card element.
 * @param {Object} movie 
 * @returns {HTMLElement}
 */
export function createMovieCard(movie, serverId = null) {
    const movieDiv = document.createElement('div');
    movieDiv.className = 'movie';

    const thumbUrl = serverId 
        ? `/remote-thumb/${serverId}/${movie.metadata_hash}`
        : `/thumb/${movie.metadata_hash}`;

    movieDiv.innerHTML = `
        <div class="poster-container">
            <img src="${thumbUrl}" alt="${movie.title}" loading="lazy">
            ${!serverId ? `<button class="play-btn" 
                                   data-hash="${movie.hash}" 
                                   data-video-codec="${movie.video_codec}" 
                                   data-audio-codec="${movie.audio_codec}" 
                                   title="Play Preview">▶</button>` : ''}
        </div>
        <div class="movie-info">
            <div class="movie-title">${movie.title} <span style="font-weight: 300; opacity: 0.7; font-size: 0.9em;">(${movie.year})</span></div>
            <div class="metadata-grid">
                <div class="metadata-label">Rating</div><div>${movie.rating || 'N/A'}</div>
                <div class="metadata-label">Genre</div><div>${movie.genre || 'N/A'}</div>
                <div class="metadata-label">Resolution</div><div>${movie.resolution || 'N/A'}</div>
                <div class="metadata-label">Codec</div><div>${movie.video_codec || 'N/A'}</div>
                <div class="metadata-label">Size</div><div>${formatSize(movie.size)}</div>
                <div class="metadata-label">Duration</div><div>${formatDuration(movie.duration)}</div>
                <div class="metadata-label">Quality Score</div><div>${(movie.quality_score || 0).toFixed(2)}</div>
                <div class="metadata-label">Critic</div><div>${(movie.critic_rating || 0).toFixed(1)}</div>
                <div class="metadata-label">Audience</div><div>${(movie.audience_rating || 0).toFixed(1)}</div>
                <div class="metadata-label">Hash</div><div style="font-family: monospace; font-size: 0.8em; word-break: break-all;">${movie.hash || 'N/A'}</div>
            </div>
        </div>
    `;

    const playBtn = movieDiv.querySelector('.play-btn');
    if (playBtn) {
        playBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            openVideoPlayer(movie.hash, movie.title, movie.video_codec, movie.audio_codec);
        });
    }

    return movieDiv;
}

/**
 * Opens a video player modal.
 * @param {string} hash 
 * @param {string} title 
 * @param {string} videoCodec
 * @param {string} audioCodec
 */
export function openVideoPlayer(hash, title, videoCodec, audioCodec) {
    let modal = document.getElementById('video-modal');
    if (!modal) {
        modal = document.createElement('div');
        modal.id = 'video-modal';
        modal.className = 'modal';
        modal.innerHTML = `
            <div class="modal-content">
                <div class="modal-header">
                    <h3 id="modal-title"></h3>
                    <span class="close-modal">&times;</span>
                </div>
                <div class="video-container">
                    <video id="video-player" controls autoplay preload="metadata" controlsList="nodownload" oncontextmenu="return false;">
                        Your browser does not support the video tag.
                    </video>
                </div>
                <div id="video-codec-info" style="padding: 10px; font-size: 0.9em; color: #ccc; background: rgba(0,0,0,0.5); border-radius: 0 0 8px 8px;">
                </div>
            </div>
        `;
        document.body.appendChild(modal);

        modal.querySelector('.close-modal').onclick = () => {
            modal.style.display = 'none';
            const video = modal.querySelector('video');
            video.pause();
            video.src = '';
        };

        window.onclick = (event) => {
            if (event.target === modal) {
                modal.querySelector('.close-modal').onclick();
            }
        };
    }

    document.getElementById('modal-title').textContent = title;
    
    const codecInfo = document.getElementById('video-codec-info');
    codecInfo.innerHTML = `<strong>Video:</strong> ${videoCodec || 'Unknown'} | <strong>Audio:</strong> ${audioCodec || 'Unknown'}`;
    
    // Add warning for common incompatible audio codecs
    const incompatibleAudio = ['dca', 'dts', 'ac3', 'truehd', 'eac3'];
    if (audioCodec && incompatibleAudio.some(c => audioCodec.toLowerCase().includes(c))) {
        codecInfo.innerHTML += `<div style="color: #ff9800; margin-top: 5px;">⚠️ Audio codec (${audioCodec}) may not be supported directly by your browser.</div>`;
    }

    const video = document.getElementById('video-player');
    video.src = `/video/${hash}`;
    modal.style.display = 'flex';
}

import { FilterPanel } from '/static/js/filter.js';

let allMovies = [];
let filterPanel = null;
let selectedServer = null;

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
    { value: 'quality_score', label: 'Quality Score' },
];

/**
 * Renders the gallery with filtered movies.
 * @param {Array} movies 
 */
function renderGallery(movies) {
    const gallery = document.getElementById('movie-gallery');
    if (!gallery) return;

    const countSpan = document.getElementById('item-count');
    if (countSpan) {
        countSpan.textContent = movies.length;
    }

    gallery.innerHTML = '';
    if (movies.length === 0) {
        gallery.innerHTML = '<p style="grid-column: 1/-1; text-align: center;">No movies match the selected filters.</p>';
        return;
    }

    movies.forEach(movie => {
        const card = createMovieCard(movie, selectedServer ? selectedServer.id : null);
        gallery.appendChild(card);
    });
}

/**
 * Handles filter and sort changes.
 * @param {FilterGroup} rootGroup 
 * @param {Object} sortConfig
 */
function handleFilterChange(rootGroup, sortConfig) {
    let filteredMovies = allMovies.filter(movie => rootGroup.apply(movie));
    
    if (sortConfig) {
        filteredMovies.sort((a, b) => {
            let valA = a[sortConfig.property];
            let valB = b[sortConfig.property];

            if (typeof valA === 'string') {
                valA = valA.toLowerCase();
                valB = valB.toLowerCase();
            }

            if (valA < valB) return sortConfig.direction === 'asc' ? -1 : 1;
            if (valA > valB) return sortConfig.direction === 'asc' ? 1 : -1;
            return 0;
        });
    }
    
    renderGallery(filteredMovies);
}

/**
 * Fetches movies from the API and populates the gallery.
 */
async function loadGallery() {
    const gallery = document.getElementById('movie-gallery');
    if (!gallery) return;

    gallery.innerHTML = `
        <div class="spinner-container">
            <div class="spinner"></div>
            <p class="loading-text">Loading movies...</p>
        </div>
    `;

    try {
        const moviesUrl = selectedServer ? `/api/servers/${selectedServer.id}/movies` : '/api/movies';
        const response = await fetch(moviesUrl);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        allMovies = await response.json();
        
        // Initialize Filter Panel
        if (!filterPanel) {
            const filterContainer = document.getElementById('filter-container');
            if (filterContainer) {
                filterPanel = new FilterPanel('filter-container', movieProperties, handleFilterChange);
            }
        }

        // Apply initial sort if filterPanel exists
        if (filterPanel) {
            handleFilterChange(filterPanel.rootGroup, {
                property: filterPanel.sortProperty,
                direction: filterPanel.sortDirection
            });
        } else {
            renderGallery(allMovies);
        }
    } catch (error) {
        console.error('Failed to load movies:', error);
        gallery.innerHTML = `
            <div class="spinner-container">
                <p style="color: #ff4d4d; text-align: center;">Error loading movies: ${error.message}</p>
            </div>
        `;
    }
}

async function loadServerPicker() {
    const picker = document.getElementById('server-picker');
    if (!picker) return;

    let serversById = new Map();

    try {
        const response = await fetch('/api/servers');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const servers = await response.json();
        serversById = new Map(servers.map(server => [String(server.id), server]));
        picker.innerHTML = '<option value="local">local</option>';
        servers.forEach(server => {
            const option = document.createElement('option');
            option.value = String(server.id);
            option.textContent = server.name || server.address || `Server ${server.id}`;
            picker.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load servers:', error);
    }

    picker.addEventListener('change', () => {
        selectedServer = picker.value === 'local' ? null : serversById.get(picker.value);
        loadGallery();
    });
}

// Initialize gallery on load
document.addEventListener('DOMContentLoaded', () => {
    loadServerPicker();
    loadGallery();
});
