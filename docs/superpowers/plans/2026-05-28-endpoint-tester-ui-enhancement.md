# Endpoint Tester UI Enhancement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enhance both admin (index.html) and user (user.html) endpoint testers with search/filter, pagination, and collapsible response boxes.

**Architecture:** Single-page enhancement using vanilla JavaScript with Tailwind CSS. All features are client-side only — no backend changes required. Search filters endpoints by name or HTTP method. Pagination splits endpoint cards into pages of configurable size. Response boxes are collapsed by default with toggle buttons.

**Tech Stack:** Vanilla JS, Tailwind CSS (CDN), existing HTML structure

---

## File Structure

- Modify: `public/index.html` — Admin endpoint tester
- Modify: `public/user.html` — User endpoint tester

---

## Task 1: Enhance index.html (Admin Endpoint Tester)

**Files:**
- Modify: `public/index.html:1-338`

### 1.1: Add search/filter UI component

Add search input and method filter buttons after the tabs section:

```html
<!-- Search & Filter -->
<div class="flex flex-wrap gap-3 mb-6 items-center">
    <div class="relative flex-1 min-w-[200px] max-w-md">
        <input type="text" id="endpointSearch" placeholder="Search endpoints..."
            class="w-full bg-black/50 border border-gray-800 rounded-lg pl-10 pr-4 py-2 text-sm text-white placeholder-gray-500"
            oninput="filterEndpoints()">
        <svg class="absolute left-3 top-2.5 w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
        </svg>
    </div>
    <div class="flex gap-2 flex-wrap">
        <button onclick="setMethodFilter('all')" id="filter-all" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-blue-600 text-white">All</button>
        <button onclick="setMethodFilter('GET')" id="filter-GET" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700">GET</button>
        <button onclick="setMethodFilter('POST')" id="filter-POST" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700">POST</button>
        <button onclick="setMethodFilter('PATCH')" id="filter-PATCH" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700">PATCH</button>
        <button onclick="setMethodFilter('DELETE')" id="filter-DELETE" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700">DELETE</button>
    </div>
    <span id="resultCount" class="text-xs text-gray-500 ml-auto">25 endpoints</span>
</div>
```

### 1.2: Add pagination UI component

Add pagination controls after the search/filter section and before endpoints container:

```html
<!-- Pagination -->
<div class="flex items-center justify-between mb-4">
    <div class="flex items-center gap-2">
        <span class="text-xs text-gray-500">Show:</span>
        <select id="pageSize" onchange="changePageSize()" class="bg-black/50 border border-gray-800 rounded px-2 py-1 text-xs text-white">
            <option value="6">6</option>
            <option value="12" selected>12</option>
            <option value="24">24</option>
            <option value="50">All</option>
        </select>
    </div>
    <div class="flex items-center gap-2">
        <button onclick="prevPage()" id="prevBtn" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed" disabled>← Prev</button>
        <span id="pageInfo" class="text-xs text-gray-500">Page 1 of 3</span>
        <button onclick="nextPage()" id="nextBtn" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700">Next →</button>
    </div>
</div>
```

### 1.3: Add collapse/expand all button

Add a toggle button in the header area:

```html
<button onclick="toggleAllResponses()" id="toggleAllBtn"
    class="px-4 py-2 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg text-sm font-medium transition-all">
    Expand All Responses
</button>
```

### 1.4: Update JavaScript state and functions

Add new state variables:

```javascript
let currentPage = 1;
let pageSize = 12;
let methodFilter = 'all';
let searchQuery = '';
let filteredEndpoints = [];
let isAllExpanded = false;
```

Add new functions after `renderEndpoints`:

```javascript
function filterEndpoints() {
    searchQuery = document.getElementById('endpointSearch').value.toLowerCase();
    currentPage = 1;
    applyFilters();
}

function setMethodFilter(method) {
    methodFilter = method;
    document.querySelectorAll('[id^="filter-"]').forEach(btn => {
        if (btn.id === 'filter-' + method) {
            btn.className = 'px-3 py-1.5 rounded-lg text-xs font-medium bg-blue-600 text-white';
        } else {
            btn.className = 'px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700';
        }
    });
    currentPage = 1;
    applyFilters();
}

function applyFilters() {
    const currentTab = document.querySelector('[class*="tab-active"]')?.id?.replace('tab-', '') || 'dashboard';
    const allEndpoints = endpoints[currentTab] || [];

    filteredEndpoints = allEndpoints.filter(ep => {
        const matchesSearch = ep.name.toLowerCase().includes(searchQuery) ||
                            ep.path.toLowerCase().includes(searchQuery);
        const matchesMethod = methodFilter === 'all' || ep.method === methodFilter;
        return matchesSearch && matchesMethod;
    });

    document.getElementById('resultCount').textContent = `${filteredEndpoints.length} endpoints`;
    renderPaginatedEndpoints(currentTab);
}

function renderPaginatedEndpoints(tab) {
    const container = document.getElementById('endpoints-container');
    const start = (currentPage - 1) * pageSize;
    const end = start + pageSize;
    const pageEndpoints = filteredEndpoints.slice(start, end);

    container.innerHTML = pageEndpoints.map((ep, idx) => {
        const actualIdx = start + idx;
        const methodColor = ep.method === 'GET' ? 'emerald' :
            ep.method === 'POST' ? 'blue' :
                ep.method === 'PATCH' ? 'amber' : 'red';
        const respId = `resp-${tab}-${actualIdx}`;
        const paramId = ep.needsParam ? `param-${tab}-${actualIdx}` : '';

        return `
            <div class="endpoint-card rounded-xl p-5">
                <div class="flex items-center justify-between mb-3">
                    <div class="flex items-center gap-3">
                        <span class="px-2.5 py-1 bg-${methodColor}-500/10 text-${methodColor}-400 rounded-md text-xs font-semibold">${ep.method}</span>
                        <h3 class="text-base font-semibold text-white">${ep.name}</h3>
                    </div>
                    <button onclick="toggleResponse('${respId}')" class="text-gray-500 hover:text-gray-300 transition-colors">
                        <svg id="icon-${respId}" class="w-5 h-5 transform rotate-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                        </svg>
                    </button>
                </div>
                <code class="text-xs text-gray-400 bg-black/50 px-3 py-1.5 rounded block mb-4">${ep.path}</code>
                ${ep.needsParam ? `
                    <input type="text" id="${paramId}" placeholder="${ep.paramName}"
                        class="w-full bg-black/50 border border-gray-800 rounded-lg px-3 py-2 text-sm text-white mb-3" />
                ` : ''}
                <button onclick="testEndpoint('${ep.method}', '${ep.path}', ${ep.needsParam ? `'${paramId}'` : 'null'}, '${respId}')"
                    class="w-full bg-blue-600 hover:bg-blue-500 text-white font-medium py-2 rounded-lg transition-all text-sm">
                    Test
                </button>
                <div id="${respId}" class="response-box rounded-lg p-4 mt-4 hidden">
                    <div class="text-xs text-gray-500 mb-2">Response:</div>
                    <pre class="text-xs font-mono text-gray-300"></pre>
                </div>
            </div>
        `;
    }).join('');

    updatePagination();
}

function updatePagination() {
    const totalPages = Math.ceil(filteredEndpoints.length / pageSize) || 1;
    document.getElementById('pageInfo').textContent = `Page ${currentPage} of ${totalPages}`;
    document.getElementById('prevBtn').disabled = currentPage === 1;
    document.getElementById('nextBtn').disabled = currentPage === totalPages;
}

function prevPage() {
    if (currentPage > 1) {
        currentPage--;
        const tab = document.querySelector('[class*="tab-active"]')?.id?.replace('tab-', '') || 'dashboard';
        renderPaginatedEndpoints(tab);
    }
}

function nextPage() {
    const totalPages = Math.ceil(filteredEndpoints.length / pageSize) || 1;
    if (currentPage < totalPages) {
        currentPage++;
        const tab = document.querySelector('[class*="tab-active"]')?.id?.replace('tab-', '') || 'dashboard';
        renderPaginatedEndpoints(tab);
    }
}

function changePageSize() {
    pageSize = parseInt(document.getElementById('pageSize').value);
    if (pageSize === 50) pageSize = 9999;
    currentPage = 1;
    const tab = document.querySelector('[class*="tab-active"]')?.id?.replace('tab-', '') || 'dashboard';
    applyFilters();
}

function toggleResponse(respId) {
    const el = document.getElementById(respId);
    const icon = document.getElementById('icon-' + respId);
    if (el.classList.contains('hidden')) {
        el.classList.remove('hidden');
        icon.classList.add('rotate-180');
    } else {
        el.classList.add('hidden');
        icon.classList.remove('rotate-180');
    }
}

function toggleAllResponses() {
    isAllExpanded = !isAllExpanded;
    const btn = document.getElementById('toggleAllBtn');
    const allResponses = document.querySelectorAll('[id^="resp-"]');

    allResponses.forEach(el => {
        if (isAllExpanded) {
            el.classList.remove('hidden');
        } else {
            el.classList.add('hidden');
        }
    });

    // Update all icons
    document.querySelectorAll('[id^="icon-resp-"]').forEach(icon => {
        if (isAllExpanded) {
            icon.classList.add('rotate-180');
        } else {
            icon.classList.remove('rotate-180');
        }
    });

    btn.textContent = isAllExpanded ? 'Collapse All Responses' : 'Expand All Responses';
    btn.className = isAllExpanded
        ? 'px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg text-sm font-medium transition-all'
        : 'px-4 py-2 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg text-sm font-medium transition-all';
}

function switchTab(tab) {
    document.querySelectorAll('[id^="tab-"]').forEach(btn => {
        btn.className = 'tab-inactive px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap';
    });
    document.getElementById('tab-' + tab).className = 'tab-active px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap';

    // Reset filters on tab change
    document.getElementById('endpointSearch').value = '';
    searchQuery = '';
    setMethodFilter('all');
    methodFilter = 'all';
    currentPage = 1;

    applyFilters();
}
```

### 1.5: Update initial render

Change `switchTab('dashboard');` at the bottom to also initialize filteredEndpoints:

```javascript
// Initialize
document.getElementById('endpointSearch').value = '';
searchQuery = '';
methodFilter = 'all';
currentPage = 1;
filteredEndpoints = endpoints['dashboard'] || [];
switchTab('dashboard');
```

---

## Task 2: Enhance user.html (User Endpoint Tester)

**Files:**
- Modify: `public/user.html:1-414`

### 2.1: Add search/filter UI component

Add after the tabs section (line ~140):

```html
<!-- Search & Filter -->
<div class="flex flex-wrap gap-3 mb-6 items-center">
    <div class="relative flex-1 min-w-[200px] max-w-md">
        <input type="text" id="endpointSearch" placeholder="Search endpoints..."
            class="w-full bg-black/50 border border-gray-800 rounded-lg pl-10 pr-4 py-2 text-sm text-white placeholder-gray-500"
            oninput="filterEndpoints()">
        <svg class="absolute left-3 top-2.5 w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
        </svg>
    </div>
    <div class="flex gap-2 flex-wrap">
        <button onclick="setMethodFilter('all')" id="filter-all" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-purple-600 text-white">All</button>
        <button onclick="setMethodFilter('GET')" id="filter-GET" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700">GET</button>
        <button onclick="setMethodFilter('POST')" id="filter-POST" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700">POST</button>
        <button onclick="setMethodFilter('PATCH')" id="filter-PATCH" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700">PATCH</button>
        <button onclick="setMethodFilter('DELETE')" id="filter-DELETE" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700">DELETE</button>
    </div>
    <span id="resultCount" class="text-xs text-gray-500 ml-auto">Loading...</span>
</div>
```

### 2.2: Add pagination UI component

Add after search/filter and before endpoints container:

```html
<!-- Pagination -->
<div class="flex items-center justify-between mb-4">
    <div class="flex items-center gap-2">
        <span class="text-xs text-gray-500">Show:</span>
        <select id="pageSize" onchange="changePageSize()" class="bg-black/50 border border-gray-800 rounded px-2 py-1 text-xs text-white">
            <option value="6">6</option>
            <option value="12" selected>12</option>
            <option value="24">24</option>
            <option value="50">All</option>
        </select>
    </div>
    <div class="flex items-center gap-2">
        <button onclick="prevPage()" id="prevBtn" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed" disabled>← Prev</button>
        <span id="pageInfo" class="text-xs text-gray-500">Page 1 of 1</span>
        <button onclick="nextPage()" id="nextBtn" class="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700">Next →</button>
    </div>
</div>
```

### 2.3: Add collapse/expand all button

Add to header area alongside the Admin Endpoints link:

```html
<button onclick="toggleAllResponses()" id="toggleAllBtn"
    class="px-4 py-2 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg text-sm font-medium transition-all">
    Expand All Responses
</button>
```

### 2.4: Update JavaScript state and functions

Add new state variables (around line 149):

```javascript
let currentPage = 1;
let pageSize = 12;
let methodFilter = 'all';
let searchQuery = '';
let filteredEndpoints = [];
let isAllExpanded = false;
```

Add new functions after `renderEndpoints` function (~line 310):

```javascript
function filterEndpoints() {
    searchQuery = document.getElementById('endpointSearch').value.toLowerCase();
    currentPage = 1;
    applyFilters();
}

function setMethodFilter(method) {
    methodFilter = method;
    document.querySelectorAll('[id^="filter-"]').forEach(btn => {
        if (btn.id === 'filter-' + method) {
            btn.className = 'px-3 py-1.5 rounded-lg text-xs font-medium bg-purple-600 text-white';
        } else {
            btn.className = 'px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700';
        }
    });
    currentPage = 1;
    applyFilters();
}

function applyFilters() {
    const currentTab = document.querySelector('[class*="tab-active"]')?.id?.replace('tab-', '') || 'profile';
    const allEndpoints = endpoints[currentTab] || [];

    filteredEndpoints = allEndpoints.filter(ep => {
        const matchesSearch = ep.name.toLowerCase().includes(searchQuery) ||
                            ep.path.toLowerCase().includes(searchQuery);
        const matchesMethod = methodFilter === 'all' || ep.method === methodFilter;
        return matchesSearch && matchesMethod;
    });

    document.getElementById('resultCount').textContent = `${filteredEndpoints.length} endpoints`;
    renderPaginatedEndpoints(currentTab);
}

function renderPaginatedEndpoints(tab) {
    const container = document.getElementById('endpoints-container');
    const start = (currentPage - 1) * pageSize;
    const end = start + pageSize;
    const pageEndpoints = filteredEndpoints.slice(start, end);

    container.innerHTML = pageEndpoints.map((ep, idx) => {
        const actualIdx = start + idx;
        const methodColor = ep.method === 'GET' ? 'emerald' :
            ep.method === 'POST' ? 'blue' :
                ep.method === 'PATCH' ? 'amber' : 'red';
        const respId = `resp-${tab}-${actualIdx}`;
        const paramId = ep.needsParam ? `param-${tab}-${actualIdx}` : '';
        const bodyId = ep.body ? `body-${tab}-${actualIdx}` : '';

        return `
            <div class="endpoint-card rounded-xl p-5">
                <div class="flex items-center justify-between mb-3">
                    <div class="flex items-center gap-3">
                        <span class="px-2.5 py-1 bg-${methodColor}-500/10 text-${methodColor}-400 rounded-md text-xs font-semibold">${ep.method}</span>
                        <h3 class="text-base font-semibold text-white">${ep.name}</h3>
                    </div>
                    <button onclick="toggleResponse('${respId}')" class="text-gray-500 hover:text-gray-300 transition-colors">
                        <svg id="icon-${respId}" class="w-5 h-5 transform rotate-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                        </svg>
                    </button>
                </div>
                <code class="text-xs text-gray-400 bg-black/50 px-3 py-1.5 rounded block mb-4">${ep.path}</code>
                ${ep.needsParam ? `
                    <input type="text" id="${paramId}" placeholder="${ep.paramName}"
                        class="w-full bg-black/50 border border-gray-800 rounded-lg px-3 py-2 text-sm text-white mb-3" />
                ` : ''}
                ${ep.body ? `
                    <textarea id="${bodyId}" rows="3"
                        class="w-full bg-black/50 border border-gray-800 rounded-lg px-3 py-2 text-xs font-mono text-white mb-3">${JSON.stringify(ep.body, null, 2)}</textarea>
                ` : ''}
                <button onclick="testEndpoint('${ep.method}', '${ep.path}', ${ep.needsParam ? `'${paramId}'` : 'null'}, ${ep.body ? `'${bodyId}'` : 'null'}, '${respId}')"
                    class="w-full bg-purple-600 hover:bg-purple-500 text-white font-medium py-2 rounded-lg transition-all text-sm">
                    Test
                </button>
                <div id="${respId}" class="response-box rounded-lg p-4 mt-4 hidden">
                    <div class="text-xs text-gray-500 mb-2">Response:</div>
                    <pre class="text-xs font-mono text-gray-300"></pre>
                </div>
            </div>
        `;
    }).join('');

    updatePagination();
}

function updatePagination() {
    const totalPages = Math.ceil(filteredEndpoints.length / pageSize) || 1;
    document.getElementById('pageInfo').textContent = `Page ${currentPage} of ${totalPages}`;
    document.getElementById('prevBtn').disabled = currentPage === 1;
    document.getElementById('nextBtn').disabled = currentPage === totalPages;
}

function prevPage() {
    if (currentPage > 1) {
        currentPage--;
        const tab = document.querySelector('[class*="tab-active"]')?.id?.replace('tab-', '') || 'profile';
        renderPaginatedEndpoints(tab);
    }
}

function nextPage() {
    const totalPages = Math.ceil(filteredEndpoints.length / pageSize) || 1;
    if (currentPage < totalPages) {
        currentPage++;
        const tab = document.querySelector('[class*="tab-active"]')?.id?.replace('tab-', '') || 'profile';
        renderPaginatedEndpoints(tab);
    }
}

function changePageSize() {
    pageSize = parseInt(document.getElementById('pageSize').value);
    if (pageSize === 50) pageSize = 9999;
    currentPage = 1;
    const tab = document.querySelector('[class*="tab-active"]')?.id?.replace('tab-', '') || 'profile';
    applyFilters();
}

function toggleResponse(respId) {
    const el = document.getElementById(respId);
    const icon = document.getElementById('icon-' + respId);
    if (el.classList.contains('hidden')) {
        el.classList.remove('hidden');
        icon.classList.add('rotate-180');
    } else {
        el.classList.add('hidden');
        icon.classList.remove('rotate-180');
    }
}

function toggleAllResponses() {
    isAllExpanded = !isAllExpanded;
    const btn = document.getElementById('toggleAllBtn');
    const allResponses = document.querySelectorAll('[id^="resp-"]');

    allResponses.forEach(el => {
        if (isAllExpanded) {
            el.classList.remove('hidden');
        } else {
            el.classList.add('hidden');
        }
    });

    document.querySelectorAll('[id^="icon-resp-"]').forEach(icon => {
        if (isAllExpanded) {
            icon.classList.add('rotate-180');
        } else {
            icon.classList.remove('rotate-180');
        }
    });

    btn.textContent = isAllExpanded ? 'Collapse All Responses' : 'Expand All Responses';
    btn.className = isAllExpanded
        ? 'px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg text-sm font-medium transition-all'
        : 'px-4 py-2 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg text-sm font-medium transition-all';
}

function switchTab(tab) {
    document.querySelectorAll('[id^="tab-"]').forEach(btn => {
        btn.className = 'tab-inactive px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap';
    });
    document.getElementById('tab-' + tab).className = 'tab-active px-4 py-2 rounded-lg text-sm font-medium whitespace-nowrap';

    document.getElementById('endpointSearch').value = '';
    searchQuery = '';
    setMethodFilter('all');
    methodFilter = 'all';
    currentPage = 1;

    applyFilters();
}
```

### 2.5: Update initial render

Change the initialization at the bottom (line ~409):

```javascript
// Initialize
document.getElementById('endpointSearch').value = '';
searchQuery = '';
methodFilter = 'all';
currentPage = 1;
filteredEndpoints = endpoints['profile'] || [];
switchTab('profile');
```

---

## Testing Instructions

1. Open `http://localhost:8080/` (admin) or `http://localhost:8080/user-test` (user)
2. Verify:
   - [ ] Search box filters endpoints by name/path as you type
   - [ ] Method filter buttons highlight selected method
   - [ ] Result count updates dynamically
   - [ ] Pagination controls show correct page info
   - [ ] Prev/Next buttons work correctly
   - [ ] Page size selector changes items per page
   - [ ] Toggle expand/collapse per card works (chevron rotates)
   - [ ] "Expand All Responses" button works
   - [ ] Tab switching resets filters and pagination
3. Test on both admin and user pages

---

## Self-Review Checklist

- [ ] Spec coverage: All 4 features implemented (tabs, search/filter, pagination, collapse)
- [ ] No placeholder code (no TODOs, no TBDs)
- [ ] Type consistency: function names match across both files
- [ ] Duplicate code: render functions are similar but use different theme colors (blue vs purple)
- [ ] Edge cases handled: empty search, no results, single page
