/**
 * Utility functions shared across multiple pages.
 */

/**
 * Retrieves a cookie value by name.
 * @param {string} name 
 * @returns {string|undefined}
 */
export function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

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
 * Parses a human-readable size string (e.g., "4gb", "10TB", "20mb") into bytes.
 * @param {string|number} size 
 * @returns {number}
 */
export function parseSize(size) {
    if (typeof size === 'number') return size;
    if (!size) return 0;

    const units = {
        'b': 1,
        'kb': 1024,
        'mb': 1024 * 1024,
        'gb': 1024 * 1024 * 1024,
        'tb': 1024 * 1024 * 1024 * 1024,
        'pb': 1024 * 1024 * 1024 * 1024 * 1024
    };

    const match = String(size).toLowerCase().match(/^(\d+(?:\.\d+)?)\s*([a-z]*)$/);
    if (!match) {
        const fallback = parseFloat(size);
        return isNaN(fallback) ? 0 : fallback;
    }

    const value = parseFloat(match[1]);
    const unit = match[2];

    if (!unit) return value; // Default to bytes if no unit

    // Improved unit matching
    let multiplier = 1;
    if (unit.startsWith('p')) multiplier = units.pb;
    else if (unit.startsWith('t')) multiplier = units.tb;
    else if (unit.startsWith('g')) multiplier = units.gb;
    else if (unit.startsWith('m')) multiplier = units.mb;
    else if (unit.startsWith('k')) multiplier = units.kb;
    
    return value * multiplier;
}

/**
 * Parses a human-readable duration string (e.g., "1h 30m", "90m") into milliseconds.
 * @param {string|number} duration 
 * @returns {number}
 */
export function parseDuration(duration) {
    if (typeof duration === 'number') return duration;
    if (!duration) return 0;

    const lower = String(duration).toLowerCase();
    
    // Check for simple numeric input first
    const numericValue = parseFloat(lower);
    if (!isNaN(numericValue) && !/[a-z]/.test(lower)) {
        return numericValue;
    }

    let totalMs = 0;
    const regex = /(\d+(?:\.\d+)?)\s*(ms|s|m|h|d|w|y)/g;
    let match;
    let found = false;

    while ((match = regex.exec(lower)) !== null) {
        found = true;
        const value = parseFloat(match[1]);
        const unit = match[2];

        switch (unit) {
            case 'ms': totalMs += value; break;
            case 's': totalMs += value * 1000; break;
            case 'm': totalMs += value * 1000 * 60; break;
            case 'h': totalMs += value * 1000 * 60 * 60; break;
            case 'd': totalMs += value * 1000 * 60 * 60 * 24; break;
            case 'w': totalMs += value * 1000 * 60 * 60 * 24 * 7; break;
            case 'y': totalMs += value * 1000 * 60 * 60 * 24 * 365; break;
        }
    }

    return found ? totalMs : (isNaN(numericValue) ? 0 : numericValue);
}

/**
 * Common fetch wrapper that includes CSRF token for state-changing methods.
 * @param {string} url 
 * @param {Object} options 
 * @returns {Promise<Response>}
 */
export async function authenticatedFetch(url, options = {}) {
    const method = (options.method || 'GET').toUpperCase();
    const stateChangingMethods = ['POST', 'PUT', 'DELETE', 'PATCH'];
    
    if (stateChangingMethods.includes(method)) {
        options.headers = {
            ...options.headers,
            'X-CSRF-Token': getCookie('csrf_token')
        };
    }
    
    return fetch(url, options);
}

/**
 * Common alert for fetch errors.
 * @param {Response} response 
 * @param {string} actionName 
 */
export async function handleFetchError(response, actionName) {
    let errorMessage = response.statusText;
    try {
        const data = await response.json();
        errorMessage = data.error || errorMessage;
    } catch (e) {
        // Not JSON
    }
    alert(`Failed to ${actionName}: ${errorMessage}`);
}
