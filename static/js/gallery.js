/**
 * Formats a size in bytes to a human-readable string (MB or GB).
 * @param {number} bytes 
 * @returns {string}
 */
export function formatSize(bytes) {
    if (bytes < 1024 * 1024 * 1024) {
        return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
    }
    return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
}

/**
 * Formats a duration in milliseconds to a human-readable string (Xh Ym).
 * @param {number} ms 
 * @returns {string}
 */
export function formatDuration(ms) {
    const durationMinutes = Math.floor(ms / (1000 * 60));
    const hours = Math.floor(durationMinutes / 60);
    const mins = durationMinutes % 60;
    return `${hours}h ${mins}m`;
}

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
                    <video id="video-player" controls autoplay preload="metadata">
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

/**
 * Fetches movies from the API and populates the gallery.
 */
async function loadGallery() {
    const gallery = document.getElementById('movie-gallery');
    if (!gallery) return;

    try {
        const response = await fetch('/dump/movies');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        // Note: The server sends gzipped JSON, but the browser handles decompression automatically
        const movies = await response.json();
        
        gallery.innerHTML = ''; // Clear loading state if any
        
        movies.forEach(movie => {
            const card = createMovieCard(movie);
            gallery.appendChild(card);
        });
    } catch (error) {
        console.error('Failed to load movies:', error);
        gallery.innerHTML = `<p style="color: red; text-align: center;">Error loading movies: ${error.message}</p>`;
    }
}

// Initialize gallery on load
document.addEventListener('DOMContentLoaded', loadGallery);
