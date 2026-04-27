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
        <img src="${thumbUrl}" alt="${movie.title}" loading="lazy">
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
            </div>
        </div>
    `;
    return movieDiv;
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
