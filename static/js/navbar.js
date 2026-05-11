import { authenticatedFetch, handleFetchError } from './utils.js';

export function initNavbar() {
    const nav = document.getElementById('main-nav');
    if (!nav) return;

    const currentPath = window.location.pathname;
    
    const links = [
        { name: 'Gallery', path: '/' },
        { name: 'Gallery', path: '/movies/gallery' },
        { name: 'Compare Servers', path: '/compare' },
        { name: 'Duplicates', path: '/duplicates' },
        { name: 'Manage Servers', path: '/add-server' },
        { name: 'Trusted Servers', path: '/trusted-servers' }
    ];

    nav.innerHTML = '';
    
    // Track added names to avoid duplicates (e.g. Gallery appearing twice)
    const addedNames = new Set();
    
    links.forEach(link => {
        if (addedNames.has(link.name)) return;
        addedNames.add(link.name);

        const a = document.createElement('a');
        a.href = link.path;
        a.textContent = link.name;
        
        // Match path - handle root and exact matches
        // We find all paths that correspond to this name and check if any match currentPath
        const allPathsForThisName = links.filter(l => l.name === link.name).map(l => l.path);
        const isActive = allPathsForThisName.includes(currentPath) || 
                         (link.name === 'Gallery' && (currentPath === '' || currentPath === '/index.html'));
        
        if (isActive) {
            a.classList.add('active');
        }
        
        nav.appendChild(a);
    });

    const logoutBtn = document.createElement('a');
    logoutBtn.href = '#';
    logoutBtn.textContent = 'Logout';
    logoutBtn.style.marginLeft = '10px';
    logoutBtn.onclick = async (e) => {
        e.preventDefault();
        const resp = await authenticatedFetch('/logout', { method: 'POST' });
        if (resp.ok) {
            window.location.href = '/login';
        } else {
            await handleFetchError(resp, 'logout');
        }
    };
    nav.appendChild(logoutBtn);
}

// Automatically initialize if the script is loaded as a module
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initNavbar);
} else {
    initNavbar();
}
