import { parseSize, parseDuration } from './utils.js';

export const FilterOperators = {
    EQUALS: '==',
    NOT_EQUALS: '!=',
    CONTAINS: 'contains',
    NOT_CONTAINS: '!contains',
    GREATER_THAN: '>',
    LESS_THAN: '<',
    STARTS_WITH: 'starts_with',
    ENDS_WITH: 'ends_with'
};

export const LogicOperators = {
    AND: 'AND',
    OR: 'OR'
};

/**
 * Represents a single filter condition.
 */
export class Filter {
    constructor(property, operator, value) {
        this.property = property;
        this.operator = operator;
        this.value = value;
    }

    apply(item) {
        let itemValue = item[this.property];
        let filterValue = this.value;

        // Special handling for size and duration properties
        if (this.property === 'size') {
            itemValue = Number(itemValue);
            filterValue = parseSize(filterValue);
        } else if (this.property === 'duration') {
            itemValue = Number(itemValue);
            filterValue = parseDuration(filterValue);
        } else if (typeof itemValue === 'number' || typeof itemValue === 'bigint') {
            const numericFilterValue = parseFloat(filterValue);
            if (!isNaN(numericFilterValue)) {
                itemValue = Number(itemValue);
                filterValue = numericFilterValue;
            }
        } else if (typeof itemValue === 'string') {
            itemValue = (itemValue || "").toLowerCase();
            filterValue = String(filterValue || "").toLowerCase();
        }

        switch (this.operator) {
            case FilterOperators.EQUALS:
                return itemValue == filterValue;
            case FilterOperators.NOT_EQUALS:
                return itemValue != filterValue;
            case FilterOperators.CONTAINS:
                return String(itemValue).includes(filterValue);
            case FilterOperators.NOT_CONTAINS:
                return !String(itemValue).includes(filterValue);
            case FilterOperators.GREATER_THAN:
                return itemValue > filterValue;
            case FilterOperators.LESS_THAN:
                return itemValue < filterValue;
            case FilterOperators.STARTS_WITH:
                return String(itemValue).startsWith(filterValue);
            case FilterOperators.ENDS_WITH:
                return String(itemValue).endsWith(filterValue);
            default:
                return true;
        }
    }

    clone() {
        return new Filter(this.property, this.operator, this.value);
    }

    serialize() {
        return {
            property: this.property,
            operator: this.operator,
            value: this.value
        };
    }
}

/**
 * Represents a group of filters or sub-groups.
 */
export class FilterGroup {
    constructor(logic = LogicOperators.AND) {
        this.logic = logic;
        this.items = []; // Can contain Filter or FilterGroup
    }

    add(item) {
        this.items.push(item);
    }

    apply(item) {
        if (this.items.length === 0) return true;

        if (this.logic === LogicOperators.AND) {
            return this.items.every(child => child.apply(item));
        } else {
            return this.items.some(child => child.apply(item));
        }
    }

    clone() {
        const cloned = new FilterGroup(this.logic);
        cloned.items = this.items.map(item => item.clone());
        return cloned;
    }

    serialize() {
        return {
            logic: this.logic,
            items: this.items.map(item => item.serialize())
        };
    }

    static deserialize(data) {
        const group = new FilterGroup(data.logic);
        if (data.items) {
            data.items.forEach(item => {
                if (item.logic) {
                    group.add(FilterGroup.deserialize(item));
                } else {
                    group.add(new Filter(item.property, item.operator, item.value));
                }
            });
        }
        return group;
    }
}

export class FilterPanel {
    constructor(containerId, properties, onFilterChange) {
        this.container = document.getElementById(containerId);
        this.properties = properties;
        this.onFilterChange = onFilterChange;
        this.rootGroup = new FilterGroup(LogicOperators.AND);
        this.sortProperty = properties[0].value;
        this.sortDirection = 'asc';
        this.activeSavedFilterId = null;
        this.activeSavedFilterName = '';
        this.preSavedFilterState = null;
        this.lastSavedState = null;
        
        this.loadFromURL();
        this.lastAppliedState = this.serializeState();
        this.init();

        window.addEventListener('popstate', () => this.handlePopState());
    }

    loadFromURL() {
        const params = new URLSearchParams(window.location.search);
        const filterData = params.get('filters');
        const sortProp = params.get('sort');
        const sortDir = params.get('dir');

        if (filterData) {
            try {
                const decoded = JSON.parse(atob(filterData));
                this.rootGroup = FilterGroup.deserialize(decoded);
            } catch (e) {
                console.error('Failed to parse filters from URL:', e);
            }
        }

        if (sortProp) this.sortProperty = sortProp;
        if (sortDir) this.sortDirection = sortDir;
    }

    updateURL() {
        const params = new URLSearchParams(window.location.search);
        const serialized = this.rootGroup.serialize();
        
        // Only add filters if there are actually filters defined
        if (serialized.items.length > 0) {
            params.set('filters', btoa(JSON.stringify(serialized)));
        } else {
            params.delete('filters');
        }

        params.set('sort', this.sortProperty);
        params.set('dir', this.sortDirection);

        const newRelativePathQuery = window.location.pathname + '?' + params.toString();
        
        // Use replaceState if the URL already has filters/sort to avoid cluttering history during a single session's tweaks
        // But for "Apply", maybe pushState is better so they can undo? 
        // User asked "encoded in the url", pushState is usually expected for "Application State"
        window.history.pushState(null, '', newRelativePathQuery);
    }

    // Helper to update only the URL when external components change (like server selection)
    static patchURL(key, value) {
        const params = new URLSearchParams(window.location.search);
        if (value) {
            params.set(key, value);
        } else {
            params.delete(key);
        }
        const newRelativePathQuery = window.location.pathname + '?' + params.toString();
        window.history.pushState(null, '', newRelativePathQuery);
    }

    static getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) return parts.pop().split(';').shift();
    }

    init() {
        this.render();
    }

    render() {
        this.container.innerHTML = '';
        this.container.classList.add('filter-panel');

        const sidebar = this.container.closest('.filter-sidebar');
        if (sidebar && !sidebar.querySelector('.filter-toggle')) {
            const toggle = document.createElement('div');
            toggle.className = 'filter-toggle';
            toggle.innerHTML = '🔍';
            toggle.title = 'Toggle Filters';
            toggle.onclick = (e) => {
                e.stopPropagation();
                sidebar.classList.toggle('collapsed');
            };
            sidebar.appendChild(toggle);

            // Add resize handle
            const handle = document.createElement('div');
            handle.className = 'resize-handle';
            sidebar.appendChild(handle);

            let isResizing = false;
            let lastX = 0;

            const startResize = (e) => {
                isResizing = true;
                lastX = e.clientX;
                handle.classList.add('resizing');
                document.addEventListener('mousemove', resize);
                document.addEventListener('mouseup', stopResize);
                document.body.style.cursor = 'ew-resize';
                e.preventDefault();
            };

            const resize = (e) => {
                if (!isResizing) return;
                const deltaX = lastX - e.clientX;
                const newWidth = sidebar.offsetWidth + deltaX;
                if (newWidth > 200 && newWidth < 1200) {
                    sidebar.style.width = `${newWidth}px`;
                    lastX = e.clientX;
                }
            };

            const stopResize = () => {
                isResizing = false;
                handle.classList.remove('resizing');
                document.removeEventListener('mousemove', resize);
                document.removeEventListener('mouseup', stopResize);
                document.body.style.cursor = '';
            };

            handle.addEventListener('mousedown', startResize);
        }

        const header = document.createElement('div');
        header.className = 'filter-header';
        header.innerHTML = '<h3>Filters</h3>';
        this.container.appendChild(header);

        const rootUI = this.createGroupUI(this.rootGroup, true);
        this.container.appendChild(rootUI);

        const divider = document.createElement('hr');
        divider.className = 'filter-divider';
        this.container.appendChild(divider);

        const sortSection = document.createElement('div');
        sortSection.className = 'sort-section';
        sortSection.innerHTML = '<h3>Sort Order</h3>';

        const sortRow = document.createElement('div');
        sortRow.className = 'sort-row';

        const sortPropSelect = document.createElement('select');
        sortPropSelect.className = 'sort-prop-select';
        this.properties.forEach(prop => {
            const opt = document.createElement('option');
            opt.value = prop.value;
            opt.textContent = prop.label;
            opt.selected = prop.value === this.sortProperty;
            sortPropSelect.appendChild(opt);
        });
        sortPropSelect.onchange = (e) => {
            this.sortProperty = e.target.value;
            this.updateApplyButton();
        };

        const sortDirSelect = document.createElement('select');
        sortDirSelect.className = 'sort-dir-select';
        sortDirSelect.innerHTML = `
            <option value="asc" ${this.sortDirection === 'asc' ? 'selected' : ''}>Ascending</option>
            <option value="desc" ${this.sortDirection === 'desc' ? 'selected' : ''}>Descending</option>
        `;
        sortDirSelect.onchange = (e) => {
            this.sortDirection = e.target.value;
            this.updateApplyButton();
        };

        sortRow.appendChild(sortPropSelect);
        sortRow.appendChild(sortDirSelect);
        sortSection.appendChild(sortRow);
        this.container.appendChild(sortSection);

        const applyBtn = document.createElement('button');
        applyBtn.id = 'apply-filters-btn';
        applyBtn.className = 'apply-filters-btn';
        applyBtn.textContent = 'Apply Changes';
        applyBtn.onclick = () => {
            this.lastAppliedState = this.serializeState();
            this.updateApplyButton();
            this.updateURL();
            this.onFilterChange(this.rootGroup, {
                property: this.sortProperty,
                direction: this.sortDirection
            });
        };
        this.container.appendChild(applyBtn);

        const resetBtn = document.createElement('button');
        resetBtn.className = 'reset-filters-btn';
        resetBtn.textContent = 'Reset';
        resetBtn.onclick = () => {
            this.rootGroup = new FilterGroup(LogicOperators.AND);
            this.sortProperty = this.properties[0].value;
            this.sortDirection = 'asc';
            this.activeSavedFilterId = null;
            this.activeSavedFilterName = '';
            this.lastSavedState = null;
            this.lastAppliedState = this.serializeState();
            this.updateURL();
            this.render();
            this.onFilterChange(this.rootGroup, {
                property: this.sortProperty,
                direction: this.sortDirection
            });
        };
        this.container.appendChild(resetBtn);

        if (this.activeSavedFilterId) {
            const updateRow = document.createElement('div');
            updateRow.className = 'update-row';
            const updateBtn = document.createElement('button');
            updateBtn.className = 'update-filters-btn';
            updateBtn.textContent = `Update "${this.activeSavedFilterName}"`;
            updateBtn.onclick = () => this.updateSavedFilter();
            
            // Disable update if no changes
            const currentState = this.serializeState();
            updateBtn.disabled = currentState === this.lastAppliedState;

            updateRow.appendChild(updateBtn);
            this.container.appendChild(updateRow);
        }

        const divider2 = document.createElement('hr');
        divider2.className = 'filter-divider';
        this.container.appendChild(divider2);

        const savedSection = document.createElement('div');
        savedSection.className = 'saved-filters-section';
        savedSection.innerHTML = '<h3>Saved Filters</h3>';

        const saveRow = document.createElement('div');
        saveRow.className = 'save-row';
        const nameInput = document.createElement('input');
        nameInput.type = 'text';
        nameInput.placeholder = 'Filter Name';
        nameInput.id = 'save-filter-name';
        const saveBtn = document.createElement('button');
        saveBtn.id = 'save-filter-btn';
        saveBtn.textContent = 'Save Current';
        saveBtn.onclick = () => this.saveCurrentFilter();
        saveRow.appendChild(nameInput);
        saveRow.appendChild(saveBtn);
        savedSection.appendChild(saveRow);

        const savedList = document.createElement('div');
        savedList.id = 'saved-filters-list';
        savedList.className = 'saved-filters-list';
        savedSection.appendChild(savedList);

        this.container.appendChild(savedSection);
        this.loadSavedFilters();

        this.updateApplyButton();
    }

    async saveCurrentFilter() {
        const nameInput = document.getElementById('save-filter-name');
        const name = nameInput.value.trim();
        if (!name) {
            alert('Please enter a name for the filter');
            return;
        }

        const state = JSON.parse(this.serializeState());
        const filterData = btoa(JSON.stringify(state));

        try {
            const resp = await fetch('/api/filters', {
                method: 'POST',
                headers: { 
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': FilterPanel.getCookie('csrf_token')
                },
                body: JSON.stringify({ name, filter_data: filterData })
            });
            if (resp.ok) {
                nameInput.value = '';
                this.loadSavedFilters();
            } else {
                const err = await resp.json();
                alert('Error saving filter: ' + err.error);
            }
        } catch (e) {
            alert('Failed to save filter: ' + e.message);
        }
    }

    async loadSavedFilters() {
        const list = document.getElementById('saved-filters-list');
        if (!list) return;

        try {
            const resp = await fetch('/api/filters');
            const filters = await resp.json();
            list.innerHTML = '';
            filters.forEach(f => {
                const item = document.createElement('div');
                item.className = 'saved-filter-item';
                if (f.id === this.activeSavedFilterId) {
                    item.classList.add('active');
                }
                
                const leftPart = document.createElement('div');
                leftPart.className = 'item-left';

                const activeMark = document.createElement('span');
                activeMark.className = 'active-mark';
                activeMark.innerHTML = f.id === this.activeSavedFilterId ? '✓' : '';
                leftPart.appendChild(activeMark);

                const nameSpan = document.createElement('span');
                nameSpan.className = 'filter-name';
                nameSpan.textContent = f.name;
                nameSpan.onclick = () => this.applySavedFilter(f);
                leftPart.appendChild(nameSpan);

                const actions = document.createElement('div');
                actions.className = 'item-actions';

                const renameBtn = document.createElement('button');
                renameBtn.innerHTML = '✎';
                renameBtn.className = 'rename-btn';
                renameBtn.title = 'Rename Filter';
                renameBtn.onclick = (e) => {
                    e.stopPropagation();
                    this.renameSavedFilter(f.id, f.name);
                };

                const deleteBtn = document.createElement('button');
                deleteBtn.textContent = '×';
                deleteBtn.className = 'delete-btn';
                deleteBtn.title = 'Delete Filter';
                deleteBtn.onclick = (e) => {
                    e.stopPropagation();
                    this.deleteSavedFilter(f.id);
                };

                actions.appendChild(renameBtn);
                actions.appendChild(deleteBtn);
                
                item.appendChild(leftPart);
                item.appendChild(actions);
                list.appendChild(item);
            });
        } catch (e) {
            console.error('Failed to load saved filters:', e);
        }
    }

    async renameSavedFilter(id, currentName) {
        const newName = prompt('Enter new name for the filter:', currentName);
        if (!newName || newName.trim() === '' || newName === currentName) return;

        try {
            // We need the data to update it, but the API might not require it if we only want to update the name
            // Actually, our PUT /api/filters/:id probably expects the full object.
            // Let's check how updateSavedFilter does it.
            
            // To be safe, fetch the filter first to get its current data
            const resp = await fetch('/api/filters');
            const filters = await resp.json();
            const filter = filters.find(f => f.id === id);
            if (!filter) return;

            const updateResp = await fetch(`/api/filters/${id}`, {
                method: 'PUT',
                headers: { 
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': FilterPanel.getCookie('csrf_token')
                },
                body: JSON.stringify({
                    name: newName.trim(),
                    filter_data: filter.filter_data
                })
            });

            if (updateResp.ok) {
                if (this.activeSavedFilterId === id) {
                    this.activeSavedFilterName = newName.trim();
                }
                this.render();
                this.loadSavedFilters();
            } else {
                const err = await updateResp.json();
                alert('Error renaming filter: ' + err.error);
            }
        } catch (e) {
            alert('Failed to rename filter: ' + e.message);
        }
    }

    applySavedFilter(filterObj) {
        try {
            if (this.activeSavedFilterId === filterObj.id) {
                // Toggling off: restore previous state
                if (this.preSavedFilterState) {
                    const decoded = JSON.parse(this.preSavedFilterState);
                    this.rootGroup = FilterGroup.deserialize(decoded.filters);
                    this.sortProperty = decoded.sort.property;
                    this.sortDirection = decoded.sort.direction;
                    this.preSavedFilterState = null;
                }
                this.activeSavedFilterId = null;
                this.activeSavedFilterName = '';
                this.lastSavedState = null;
            } else {
                // Toggling on (or switching): 
                // If we aren't already in a saved filter, save the current state as "pre-selection"
                if (!this.activeSavedFilterId) {
                    this.preSavedFilterState = this.serializeState();
                }

                const decoded = JSON.parse(atob(filterObj.filter_data));
                this.rootGroup = FilterGroup.deserialize(decoded.filters);
                this.sortProperty = decoded.sort.property;
                this.sortDirection = decoded.sort.direction;
                
                this.activeSavedFilterId = filterObj.id;
                this.activeSavedFilterName = filterObj.name;
                this.lastSavedState = this.serializeState();
            }
            
            this.lastAppliedState = this.serializeState();
            this.updateURL();
            this.render();
            this.onFilterChange(this.rootGroup, {
                property: this.sortProperty,
                direction: this.sortDirection
            });
        } catch (e) {
            console.error('Failed to apply saved filter:', e);
            alert('Failed to apply saved filter');
        }
    }

    async deleteSavedFilter(id) {
        if (!confirm('Are you sure you want to delete this saved filter?')) return;
        try {
            const resp = await fetch(`/api/filters/${id}`, { 
                method: 'DELETE',
                headers: {
                    'X-CSRF-Token': FilterPanel.getCookie('csrf_token')
                }
            });
            if (resp.ok) {
                if (this.activeSavedFilterId === id) {
                    this.activeSavedFilterId = null;
                    this.activeSavedFilterName = '';
                    this.render();
                }
                this.loadSavedFilters();
            }
        } catch (e) {
            console.error('Failed to delete filter:', e);
        }
    }

    async updateSavedFilter() {
        if (!this.activeSavedFilterId) return;

        const state = JSON.parse(this.serializeState());
        const filterData = btoa(JSON.stringify(state));

        try {
            const resp = await fetch(`/api/filters/${this.activeSavedFilterId}`, {
                method: 'PUT',
                headers: { 
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': FilterPanel.getCookie('csrf_token')
                },
                body: JSON.stringify({ 
                    name: this.activeSavedFilterName, 
                    filter_data: filterData 
                })
            });
            if (resp.ok) {
                this.lastSavedState = this.serializeState();
                this.lastAppliedState = this.serializeState();
                this.render();
                this.loadSavedFilters();
            } else {
                const err = await resp.json();
                alert('Error updating filter: ' + err.error);
            }
        } catch (e) {
            alert('Failed to update filter: ' + e.message);
        }
    }

    serializeState() {
        return JSON.stringify({
            filters: this.rootGroup.serialize(),
            sort: {
                property: this.sortProperty,
                direction: this.sortDirection
            }
        });
    }

    updateApplyButton() {
        const applyBtn = document.getElementById('apply-filters-btn');
        const saveBtn = document.getElementById('save-filter-btn');
        if (!applyBtn) return;

        const currentState = this.serializeState();
        const isModifiedFromApplied = currentState !== this.lastAppliedState;
        applyBtn.disabled = !isModifiedFromApplied;
        if (saveBtn) saveBtn.disabled = false; // Always allow saving the current visible state

        const updateBtn = this.container.querySelector('.update-filters-btn');
        if (updateBtn) {
            const isModifiedFromSaved = currentState !== this.lastSavedState;
            updateBtn.disabled = !isModifiedFromSaved;
        }
    }

    // Listens for popstate to handle browser back/forward
    handlePopState() {
        this.loadFromURL();
        this.render();
        this.lastAppliedState = this.serializeState();
        this.onFilterChange(this.rootGroup, {
            property: this.sortProperty,
            direction: this.sortDirection
        });
    }

    createGroupUI(group, isRoot = false) {
        const groupDiv = document.createElement('div');
        groupDiv.className = 'filter-group-ui';
        if (isRoot) groupDiv.classList.add('root-group');

        const controls = document.createElement('div');
        controls.className = 'group-controls';

        const logicSelect = document.createElement('select');
        logicSelect.innerHTML = `
            <option value="${LogicOperators.AND}" ${group.logic === LogicOperators.AND ? 'selected' : ''}>AND</option>
            <option value="${LogicOperators.OR}" ${group.logic === LogicOperators.OR ? 'selected' : ''}>OR</option>
        `;
        logicSelect.onchange = (e) => {
            group.logic = e.target.value;
            this.updateApplyButton();
        };
        controls.appendChild(logicSelect);

        const addFilterBtn = document.createElement('button');
        addFilterBtn.textContent = '+ Filter';
        addFilterBtn.onclick = () => {
            group.add(new Filter(this.properties[0].value, FilterOperators.CONTAINS, ''));
            this.renderGroupItems(group, itemsContainer);
            this.updateApplyButton();
        };
        controls.appendChild(addFilterBtn);

        const addGroupBtn = document.createElement('button');
        addGroupBtn.textContent = '+ Group';
        addGroupBtn.onclick = () => {
            group.add(new FilterGroup(LogicOperators.AND));
            this.renderGroupItems(group, itemsContainer);
            this.updateApplyButton();
        };
        controls.appendChild(addGroupBtn);

        if (!isRoot) {
            const removeBtn = document.createElement('button');
            removeBtn.textContent = '×';
            removeBtn.className = 'remove-btn';
            removeBtn.onclick = () => {
                // This is a bit tricky, we need to find the parent and remove this group.
                // For simplicity in this implementation, we re-render everything.
                this.removeItemFromParent(this.rootGroup, group);
                this.render();
                this.updateApplyButton();
            };
            controls.appendChild(removeBtn);
        }

        groupDiv.appendChild(controls);

        const itemsContainer = document.createElement('div');
        itemsContainer.className = 'group-items';
        groupDiv.appendChild(itemsContainer);

        this.renderGroupItems(group, itemsContainer);

        return groupDiv;
    }

    renderGroupItems(group, container) {
        container.innerHTML = '';
        group.items.forEach((item, index) => {
            if (item instanceof Filter) {
                container.appendChild(this.createFilterUI(item, group));
            } else if (item instanceof FilterGroup) {
                container.appendChild(this.createGroupUI(item));
            }
        });
    }

    createFilterUI(filter, parentGroup) {
        const filterDiv = document.createElement('div');
        filterDiv.className = 'filter-row';

        const propSelect = document.createElement('select');
        this.properties.forEach(prop => {
            const opt = document.createElement('option');
            opt.value = prop.value;
            opt.textContent = prop.label;
            if (filter.property === prop.value) opt.selected = true;
            propSelect.appendChild(opt);
        });
        propSelect.onchange = (e) => {
            filter.property = e.target.value;
            this.updateApplyButton();
        };

        const opSelect = document.createElement('select');
        Object.values(FilterOperators).forEach(op => {
            const opt = document.createElement('option');
            opt.value = op;
            opt.textContent = op;
            if (filter.operator === op) opt.selected = true;
            opSelect.appendChild(opt);
        });
        opSelect.onchange = (e) => {
            filter.operator = e.target.value;
            this.updateApplyButton();
        };

        const valInput = document.createElement('input');
        valInput.type = 'text';
        valInput.value = filter.value;
        valInput.oninput = (e) => {
            filter.value = e.target.value;
            this.updateApplyButton();
        };

        const removeBtn = document.createElement('button');
        removeBtn.textContent = '×';
        removeBtn.className = 'remove-btn';
        removeBtn.onclick = () => {
            parentGroup.items = parentGroup.items.filter(i => i !== filter);
            this.render();
            this.updateApplyButton();
        };

        filterDiv.appendChild(propSelect);
        filterDiv.appendChild(opSelect);
        filterDiv.appendChild(valInput);
        filterDiv.appendChild(removeBtn);

        return filterDiv;
    }

    removeItemFromParent(parent, target) {
        const index = parent.items.indexOf(target);
        if (index > -1) {
            parent.items.splice(index, 1);
            return true;
        }
        for (let item of parent.items) {
            if (item instanceof FilterGroup) {
                if (this.removeItemFromParent(item, target)) return true;
            }
        }
        return false;
    }
}
