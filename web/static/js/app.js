// Simple Memory Dashboard App
const GRAPHQL_ENDPOINT = '/graphql';

let allMemories = [];
let selectedMemory = null;
let memoryGraph = new Map(); // For storing relationships

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    loadRepositories();
    loadRecentMemories();
    setupEventListeners();
});

function setupEventListeners() {
    document.getElementById('repository').addEventListener('change', loadRecentMemories);
    document.getElementById('recency').addEventListener('change', loadRecentMemories);
    document.getElementById('type').addEventListener('change', loadRecentMemories);
    document.getElementById('searchQuery').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') searchMemories();
    });
}

// GraphQL query helper
async function graphQLQuery(query, variables = {}) {
    try {
        const response = await fetch(GRAPHQL_ENDPOINT, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ query, variables })
        });
        
        const data = await response.json();
        if (data.errors) {
            console.error('GraphQL errors:', data.errors);
            throw new Error(data.errors[0].message);
        }
        return data.data;
    } catch (error) {
        console.error('GraphQL query failed:', error);
        throw error;
    }
}

// Load repositories for dropdown
async function loadRepositories() {
    // For now, we'll use static repos. In future, could query from API
    const repos = ['mcp-memory', 'my-project', 'another-project'];
    const select = document.getElementById('repository');
    
    repos.forEach(repo => {
        const option = document.createElement('option');
        option.value = repo;
        option.textContent = repo;
        select.appendChild(option);
    });
}

// Load recent memories
async function loadRecentMemories() {
    const repository = document.getElementById('repository').value || "mcp-memory";
    const recency = document.getElementById('recency').value;
    const type = document.getElementById('type').value;
    
    // Use listChunks for loading without search query
    const query = `
        query ListChunks($repository: String!, $limit: Int!) {
            listChunks(repository: $repository, limit: $limit) {
                id
                sessionId
                repository
                timestamp
                content
                summary
                type
                tags
            }
        }
    `;
    
    const variables = {
        repository: repository,
        limit: 50
    };
    
    try {
        showLoading('memoryList');
        const data = await graphQLQuery(query, variables);
        // Wrap in search-like format for compatibility
        const wrappedData = data.listChunks.map(chunk => ({
            chunk: chunk,
            score: 1.0
        }));
        displayMemories(wrappedData);
        updateStats(wrappedData.length);
    } catch (error) {
        showError('memoryList', 'Failed to load memories');
    }
}

// Search memories
async function searchMemories() {
    const searchQuery = document.getElementById('searchQuery').value;
    if (!searchQuery.trim()) {
        loadRecentMemories();
        return;
    }
    
    const repository = document.getElementById('repository').value;
    const recency = document.getElementById('recency').value;
    
    const query = `
        query SearchMemories($input: MemoryQueryInput!) {
            search(input: $input) {
                chunks {
                    chunk {
                        id
                        sessionId
                        repository
                        timestamp
                        content
                        summary
                        type
                        tags
                    }
                    score
                }
            }
        }
    `;
    
    const variables = {
        input: {
            query: searchQuery,
            recency: recency,
            limit: 20
        }
    };
    
    if (repository) {
        variables.input.repository = repository;
    }
    
    try {
        showLoading('memoryList');
        const data = await graphQLQuery(query, variables);
        displayMemories(data.search.chunks);
        updateStats(data.search.chunks.length);
    } catch (error) {
        showError('memoryList', 'Search failed');
    }
}

// Display memories in the list
function displayMemories(memories) {
    const container = document.getElementById('memoryList');
    container.innerHTML = '';
    
    if (memories.length === 0) {
        container.innerHTML = '<div class="placeholder">No memories found</div>';
        return;
    }
    
    allMemories = memories;
    
    memories.forEach((item, index) => {
        const memory = item.chunk;
        const div = document.createElement('div');
        div.className = 'memory-item';
        div.dataset.index = index;
        
        const date = new Date(memory.timestamp).toLocaleDateString();
        const summary = memory.summary || memory.content.substring(0, 100) + '...';
        
        div.innerHTML = `
            <h3>${getTypeIcon(memory.type)} ${summary}</h3>
            <div class="meta">
                ${memory.repository || 'global'} • ${date} • Score: ${(item.score * 100).toFixed(0)}%
            </div>
            ${memory.tags && memory.tags.length > 0 ? `
                <div class="tags">
                    ${memory.tags.map(tag => `<span class="tag">${tag}</span>`).join('')}
                </div>
            ` : ''}
        `;
        
        div.addEventListener('click', () => selectMemory(index));
        container.appendChild(div);
    });
    
    // Select first memory by default
    if (memories.length > 0) {
        selectMemory(0);
    }
}

// Select and display memory details
function selectMemory(index) {
    const memoryData = allMemories[index];
    if (!memoryData) return;
    
    selectedMemory = memoryData;
    const memory = memoryData.chunk;
    
    // Update selection in list
    document.querySelectorAll('.memory-item').forEach(item => {
        item.classList.remove('selected');
    });
    document.querySelector(`[data-index="${index}"]`).classList.add('selected');
    
    // Display details
    const detailContainer = document.getElementById('memoryDetail');
    detailContainer.innerHTML = `
        <div class="memory-detail-content">
            <div class="field">
                <div class="label">Score</div>
                <div class="value">
                    <span class="score">${(memoryData.score * 100).toFixed(1)}%</span>
                </div>
            </div>
            
            <div class="field">
                <div class="label">Type</div>
                <div class="value">${getTypeIcon(memory.type)} ${memory.type}</div>
            </div>
            
            <div class="field">
                <div class="label">Repository</div>
                <div class="value">${memory.repository || '_global'}</div>
            </div>
            
            <div class="field">
                <div class="label">Session ID</div>
                <div class="value">${memory.sessionId}</div>
            </div>
            
            <div class="field">
                <div class="label">Timestamp</div>
                <div class="value">${new Date(memory.timestamp).toLocaleString()}</div>
            </div>
            
            ${memory.tags && memory.tags.length > 0 ? `
                <div class="field">
                    <div class="label">Tags</div>
                    <div class="value">
                        ${memory.tags.map(tag => `<span class="tag">${tag}</span>`).join('')}
                    </div>
                </div>
            ` : ''}
            
            ${memory.summary ? `
                <div class="field">
                    <div class="label">Summary</div>
                    <div class="value">${memory.summary}</div>
                </div>
            ` : ''}
            
            <div class="field">
                <div class="label">Content</div>
                <div class="value">${memory.content}</div>
            </div>
        </div>
    `;
    
    // Show trace buttons
    document.getElementById('traceButtons').style.display = 'flex';
    
    // Update visualization
    updateVisualization();
}

// Simple visualization using canvas
function updateVisualization() {
    const canvas = document.getElementById('graphCanvas');
    const ctx = canvas.getContext('2d');
    
    // Set canvas size
    canvas.width = canvas.offsetWidth;
    canvas.height = canvas.offsetHeight;
    
    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    if (!selectedMemory) return;
    
    // Simple visualization: show selected memory in center with related memories around it
    const centerX = canvas.width / 2;
    const centerY = canvas.height / 2;
    const radius = 150;
    
    // Draw center node (selected memory)
    drawNode(ctx, centerX, centerY, selectedMemory.chunk, '#3498db', true);
    
    // Draw related memories (same session or similar tags)
    const related = findRelatedMemories(selectedMemory.chunk);
    const angleStep = (2 * Math.PI) / Math.max(related.length, 1);
    
    related.forEach((memory, index) => {
        const angle = index * angleStep;
        const x = centerX + radius * Math.cos(angle);
        const y = centerY + radius * Math.sin(angle);
        
        // Draw connection
        ctx.beginPath();
        ctx.moveTo(centerX, centerY);
        ctx.lineTo(x, y);
        ctx.strokeStyle = '#ddd';
        ctx.lineWidth = 1;
        ctx.stroke();
        
        // Draw node
        drawNode(ctx, x, y, memory.chunk, '#95a5a6', false);
    });
}

function drawNode(ctx, x, y, memory, color, isSelected) {
    const nodeRadius = isSelected ? 30 : 20;
    
    // Draw circle
    ctx.beginPath();
    ctx.arc(x, y, nodeRadius, 0, 2 * Math.PI);
    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = isSelected ? '#2c3e50' : '#bdc3c7';
    ctx.lineWidth = isSelected ? 3 : 2;
    ctx.stroke();
    
    // Draw icon
    ctx.fillStyle = 'white';
    ctx.font = isSelected ? '20px sans-serif' : '14px sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(getTypeIcon(memory.type), x, y);
    
    // Draw label
    if (isSelected) {
        ctx.fillStyle = '#2c3e50';
        ctx.font = '12px sans-serif';
        const label = memory.summary || memory.content.substring(0, 30) + '...';
        ctx.fillText(label.substring(0, 30), x, y + nodeRadius + 15);
    }
}

function findRelatedMemories(memory) {
    // Find memories from same session or with overlapping tags
    return allMemories.filter(item => {
        const other = item.chunk;
        if (other.id === memory.id) return false;
        
        // Same session
        if (other.sessionId === memory.sessionId) return true;
        
        // Overlapping tags
        if (memory.tags && other.tags) {
            const overlap = memory.tags.some(tag => other.tags.includes(tag));
            if (overlap) return true;
        }
        
        return false;
    }).slice(0, 6); // Limit to 6 related memories for visual clarity
}

// Helper functions
function getTypeIcon(type) {
    const icons = {
        'problem': '🐛',
        'solution': '✅',
        'architecture_decision': '🏗️',
        'session_summary': '📋',
        'code_change': '💻',
        'discussion': '💬',
        'analysis': '📊',
        'verification': '✓'
    };
    return icons[type] || '📝';
}

function updateStats(count) {
    document.getElementById('stats').innerHTML = `
        <span>Showing ${count} memories</span>
    `;
}

function showLoading(elementId) {
    document.getElementById(elementId).innerHTML = '<div class="loading">Loading...</div>';
}

// Trace functions
async function traceSession() {
    if (!selectedMemory || !selectedMemory.chunk) return;
    
    const sessionId = selectedMemory.chunk.sessionId;
    const query = `
        query TraceSession($sessionId: String!) {
            traceSession(sessionId: $sessionId) {
                id
                sessionId
                repository
                timestamp
                content
                summary
                type
                tags
            }
        }
    `;
    
    try {
        showLoading('memoryList');
        const data = await graphQLQuery(query, { sessionId });
        
        // Display traced memories
        const wrappedData = data.traceSession.map(chunk => ({
            chunk: chunk,
            score: 1.0
        }));
        displayMemories(wrappedData);
        updateStats(wrappedData.length);
        
        // Update visualization to show session timeline
        visualizeTimeline(data.traceSession);
    } catch (error) {
        showError('memoryList', 'Failed to trace session');
    }
}

async function traceRelated() {
    if (!selectedMemory || !selectedMemory.chunk) return;
    
    const chunkId = selectedMemory.chunk.id;
    const query = `
        query TraceRelated($chunkId: String!, $depth: Int) {
            traceRelated(chunkId: $chunkId, depth: $depth) {
                id
                sessionId
                repository
                timestamp
                content
                summary
                type
                tags
            }
        }
    `;
    
    try {
        showLoading('memoryList');
        const data = await graphQLQuery(query, { chunkId, depth: 2 });
        
        // Display related memories
        const wrappedData = data.traceRelated.map(chunk => ({
            chunk: chunk,
            score: 1.0
        }));
        displayMemories(wrappedData);
        updateStats(wrappedData.length);
        
        // Update visualization to show relationship graph
        visualizeRelationships(data.traceRelated);
    } catch (error) {
        showError('memoryList', 'Failed to find related memories');
    }
}

// Timeline visualization for session trace
function visualizeTimeline(chunks) {
    const canvas = document.getElementById('graphCanvas');
    const ctx = canvas.getContext('2d');
    
    canvas.width = canvas.offsetWidth;
    canvas.height = canvas.offsetHeight;
    
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    if (chunks.length === 0) return;
    
    // Sort by timestamp
    chunks.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
    
    // Calculate positions
    const margin = 50;
    const nodeRadius = 20;
    const lineY = canvas.height / 2;
    const spacing = (canvas.width - 2 * margin) / (chunks.length - 1);
    
    // Draw timeline
    ctx.strokeStyle = '#ddd';
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.moveTo(margin, lineY);
    ctx.lineTo(canvas.width - margin, lineY);
    ctx.stroke();
    
    // Draw nodes
    chunks.forEach((chunk, i) => {
        const x = margin + i * spacing;
        const y = lineY;
        
        // Draw connection lines
        if (i > 0) {
            ctx.strokeStyle = '#3498db';
            ctx.lineWidth = 2;
            ctx.beginPath();
            ctx.moveTo(margin + (i-1) * spacing + nodeRadius, y);
            ctx.lineTo(x - nodeRadius, y);
            ctx.stroke();
        }
        
        // Draw node
        ctx.fillStyle = getTypeColor(chunk.type);
        ctx.beginPath();
        ctx.arc(x, y, nodeRadius, 0, 2 * Math.PI);
        ctx.fill();
        
        // Draw icon
        ctx.fillStyle = 'white';
        ctx.font = '16px Arial';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(getTypeIcon(chunk.type), x, y);
        
        // Draw timestamp
        ctx.fillStyle = '#666';
        ctx.font = '10px Arial';
        ctx.textAlign = 'center';
        const date = new Date(chunk.timestamp);
        ctx.fillText(date.toLocaleTimeString(), x, y + 35);
        
        // Draw summary
        if (chunk.summary) {
            ctx.font = '12px Arial';
            const summary = chunk.summary.substring(0, 20) + '...';
            ctx.fillText(summary, x, y - 35);
        }
    });
}

// Relationship visualization for related trace
function visualizeRelationships(chunks) {
    const canvas = document.getElementById('graphCanvas');
    const ctx = canvas.getContext('2d');
    
    canvas.width = canvas.offsetWidth;
    canvas.height = canvas.offsetHeight;
    
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    if (chunks.length === 0) return;
    
    // Layout nodes in a circle
    const centerX = canvas.width / 2;
    const centerY = canvas.height / 2;
    const radius = Math.min(canvas.width, canvas.height) * 0.3;
    const nodeRadius = 25;
    
    const positions = chunks.map((chunk, i) => {
        const angle = (i / chunks.length) * 2 * Math.PI - Math.PI / 2;
        return {
            x: centerX + radius * Math.cos(angle),
            y: centerY + radius * Math.sin(angle),
            chunk: chunk
        };
    });
    
    // Draw connections (simplified - connect adjacent nodes)
    ctx.strokeStyle = '#ddd';
    ctx.lineWidth = 1;
    positions.forEach((pos, i) => {
        const next = positions[(i + 1) % positions.length];
        ctx.beginPath();
        ctx.moveTo(pos.x, pos.y);
        ctx.lineTo(next.x, next.y);
        ctx.stroke();
    });
    
    // Draw nodes
    positions.forEach((pos, i) => {
        // Highlight the center node (original chunk)
        const isCenter = i === 0;
        
        // Draw node
        ctx.fillStyle = isCenter ? '#e74c3c' : getTypeColor(pos.chunk.type);
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, nodeRadius, 0, 2 * Math.PI);
        ctx.fill();
        
        // Draw border for center node
        if (isCenter) {
            ctx.strokeStyle = '#c0392b';
            ctx.lineWidth = 3;
            ctx.stroke();
        }
        
        // Draw icon
        ctx.fillStyle = 'white';
        ctx.font = '18px Arial';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(getTypeIcon(pos.chunk.type), pos.x, pos.y);
        
        // Draw label
        ctx.fillStyle = '#333';
        ctx.font = '12px Arial';
        const label = pos.chunk.type;
        ctx.fillText(label, pos.x, pos.y + nodeRadius + 15);
    });
}

function getTypeColor(type) {
    const colors = {
        'problem': '#e74c3c',
        'solution': '#2ecc71',
        'architecture_decision': '#3498db',
        'session_summary': '#f39c12',
        'code_change': '#9b59b6',
        'discussion': '#1abc9c',
        'analysis': '#34495e',
        'verification': '#27ae60'
    };
    return colors[type] || '#95a5a6';
}

function showError(elementId, message) {
    document.getElementById(elementId).innerHTML = `<div class="placeholder">${message}</div>`;
}