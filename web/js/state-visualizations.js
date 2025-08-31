// State File Visualization Components for DriftMgr

class StateVisualizations {
    constructor() {
        this.charts = {};
        this.graphInstances = {};
        this.colors = {
            healthy: '#10b981',
            warning: '#f59e0b',
            critical: '#ef4444',
            unknown: '#6b7280',
            managed: '#3b82f6',
            outOfBand: '#dc2626',
            conflict: '#f97316',
            fresh: '#22c55e',
            recent: '#84cc16',
            stale: '#eab308',
            abandoned: '#dc2626'
        };
    }

    // Initialize all visualizations
    init() {
        // Initialize D3.js for complex visualizations
        if (typeof d3 === 'undefined') {
            this.loadD3().then(() => {
                this.initializeVisualizations();
            });
        } else {
            this.initializeVisualizations();
        }
    }

    // Load D3.js dynamically
    loadD3() {
        return new Promise((resolve) => {
            const script = document.createElement('script');
            script.src = 'https://d3js.org/d3.v7.min.js';
            script.onload = resolve;
            document.head.appendChild(script);
        });
    }

    // Initialize visualization components
    initializeVisualizations() {
        // These will be called when specific views are opened
        console.log('State visualizations ready');
    }

    // Create State File Galaxy View
    createStateGalaxyView(containerId, stateFiles) {
        const container = document.getElementById(containerId);
        if (!container || !d3) return;

        // Clear existing content
        container.innerHTML = '';

        const width = container.clientWidth;
        const height = 600;

        const svg = d3.select(`#${containerId}`)
            .append('svg')
            .attr('width', width)
            .attr('height', height);

        // Create force simulation
        const simulation = d3.forceSimulation()
            .force('charge', d3.forceManyBody().strength(-300))
            .force('center', d3.forceCenter(width / 2, height / 2))
            .force('collision', d3.forceCollide().radius(d => d.radius + 2));

        // Process state files into nodes
        const nodes = stateFiles.map(state => ({
            id: state.id,
            name: state.name,
            type: 'state',
            radius: Math.sqrt(state.resource_count) * 5 + 10,
            health: state.health.status,
            resourceCount: state.resource_count,
            backend: state.type,
            age: state.health.age
        }));

        // Add resource nodes for each state file
        stateFiles.forEach(state => {
            if (state.resources && state.resources.length > 0) {
                state.resources.slice(0, 20).forEach(resource => {
                    nodes.push({
                        id: resource.id,
                        name: resource.name,
                        type: 'resource',
                        parent: state.id,
                        radius: 5,
                        resourceType: resource.type,
                        status: resource.status
                    });
                });
            }
        });

        // Create links between states and their resources
        const links = [];
        nodes.forEach(node => {
            if (node.type === 'resource' && node.parent) {
                links.push({
                    source: node.parent,
                    target: node.id
                });
            }
        });

        // Add links to simulation
        simulation
            .nodes(nodes)
            .force('link', d3.forceLink(links).id(d => d.id).distance(100));

        // Create link elements
        const link = svg.append('g')
            .selectAll('line')
            .data(links)
            .enter().append('line')
            .attr('stroke', '#999')
            .attr('stroke-opacity', 0.6)
            .attr('stroke-width', 1);

        // Create node groups
        const node = svg.append('g')
            .selectAll('g')
            .data(nodes)
            .enter().append('g')
            .call(d3.drag()
                .on('start', dragstarted)
                .on('drag', dragged)
                .on('end', dragended));

        // Add circles for nodes
        node.append('circle')
            .attr('r', d => d.radius)
            .attr('fill', d => {
                if (d.type === 'state') {
                    return this.colors[d.health] || this.colors.unknown;
                } else {
                    return d.status === 'drifted' ? this.colors.conflict : this.colors.managed;
                }
            })
            .attr('stroke', '#fff')
            .attr('stroke-width', 2);

        // Add labels for state files
        node.filter(d => d.type === 'state')
            .append('text')
            .text(d => d.name)
            .attr('x', 0)
            .attr('y', d => d.radius + 15)
            .attr('text-anchor', 'middle')
            .attr('font-size', '12px')
            .attr('fill', '#333');

        // Add tooltips
        node.append('title')
            .text(d => {
                if (d.type === 'state') {
                    return `${d.name}\nResources: ${d.resourceCount}\nHealth: ${d.health}\nBackend: ${d.backend}`;
                } else {
                    return `${d.name}\nType: ${d.resourceType}\nStatus: ${d.status}`;
                }
            });

        // Update positions on simulation tick
        simulation.on('tick', () => {
            link
                .attr('x1', d => d.source.x)
                .attr('y1', d => d.source.y)
                .attr('x2', d => d.target.x)
                .attr('y2', d => d.target.y);

            node.attr('transform', d => `translate(${d.x},${d.y})`);
        });

        // Drag functions
        function dragstarted(event, d) {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            d.fx = d.x;
            d.fy = d.y;
        }

        function dragged(event, d) {
            d.fx = event.x;
            d.fy = event.y;
        }

        function dragended(event, d) {
            if (!event.active) simulation.alphaTarget(0);
            d.fx = null;
            d.fy = null;
        }
    }

    // Create Sankey Diagram for Resource Flow
    createSankeyDiagram(containerId, perspective) {
        const container = document.getElementById(containerId);
        if (!container || !d3) return;

        container.innerHTML = '';

        const width = container.clientWidth;
        const height = 500;
        const margin = { top: 20, right: 20, bottom: 20, left: 20 };

        // Process data for Sankey
        const nodes = [];
        const links = [];

        // Add state file as source
        nodes.push({ id: 'state', name: 'State File' });

        // Add providers
        const providers = {};
        perspective.managed_resources.forEach(resource => {
            if (!providers[resource.provider]) {
                providers[resource.provider] = { count: 0, id: `provider_${resource.provider}` };
                nodes.push({ id: providers[resource.provider].id, name: resource.provider.toUpperCase() });
            }
            providers[resource.provider].count++;
        });

        // Add resource types
        const resourceTypes = {};
        perspective.managed_resources.forEach(resource => {
            const typeKey = `${resource.provider}_${resource.type}`;
            if (!resourceTypes[typeKey]) {
                resourceTypes[typeKey] = { count: 0, id: `type_${typeKey}` };
                nodes.push({ id: resourceTypes[typeKey].id, name: resource.type });
            }
            resourceTypes[typeKey].count++;
        });

        // Create links from state to providers
        Object.values(providers).forEach(provider => {
            links.push({
                source: 'state',
                target: provider.id,
                value: provider.count
            });
        });

        // Create links from providers to resource types
        perspective.managed_resources.forEach(resource => {
            const providerId = `provider_${resource.provider}`;
            const typeId = `type_${resource.provider}_${resource.type}`;
            const existingLink = links.find(l => l.source === providerId && l.target === typeId);
            if (existingLink) {
                existingLink.value++;
            } else {
                links.push({
                    source: providerId,
                    target: typeId,
                    value: 1
                });
            }
        });

        // Would implement full Sankey here with d3-sankey plugin
        // For now, create a simple flow visualization
        const svg = d3.select(`#${containerId}`)
            .append('svg')
            .attr('width', width)
            .attr('height', height);

        // Create Sankey diagram data structure
        const nodes = [];
        const links = [];
        const nodeMap = new Map();
        let nodeIndex = 0;

        // Create nodes for providers, resource types, and states
        const providers = new Set();
        const resourceTypes = new Set();
        const states = new Set();

        stateFiles.forEach(state => {
            if (state.resources) {
                state.resources.forEach(resource => {
                    providers.add(resource.provider || 'unknown');
                    resourceTypes.add(resource.type || 'unknown');
                    states.add(resource.state || 'active');
                });
            }
        });

        // Add provider nodes (source)
        providers.forEach(provider => {
            nodeMap.set(`provider-${provider}`, nodeIndex);
            nodes.push({ 
                id: nodeIndex++, 
                name: provider, 
                category: 'provider',
                color: this.getProviderColor(provider)
            });
        });

        // Add resource type nodes (middle)
        resourceTypes.forEach(type => {
            nodeMap.set(`type-${type}`, nodeIndex);
            nodes.push({ 
                id: nodeIndex++, 
                name: type, 
                category: 'resource',
                color: '#64748b'
            });
        });

        // Add state nodes (target)
        states.forEach(state => {
            nodeMap.set(`state-${state}`, nodeIndex);
            nodes.push({ 
                id: nodeIndex++, 
                name: state, 
                category: 'state',
                color: this.getStateColor(state)
            });
        });

        // Create links between nodes
        stateFiles.forEach(state => {
            if (state.resources) {
                // Count resources by provider->type->state
                const flowMap = new Map();
                
                state.resources.forEach(resource => {
                    const provider = resource.provider || 'unknown';
                    const type = resource.type || 'unknown';
                    const resourceState = resource.state || 'active';
                    
                    const providerTypeKey = `${provider}->${type}`;
                    const typeStateKey = `${type}->${resourceState}`;
                    
                    flowMap.set(providerTypeKey, (flowMap.get(providerTypeKey) || 0) + 1);
                    flowMap.set(typeStateKey, (flowMap.get(typeStateKey) || 0) + 1);
                });

                // Add provider to type links
                providers.forEach(provider => {
                    resourceTypes.forEach(type => {
                        const key = `${provider}->${type}`;
                        const value = flowMap.get(key);
                        if (value > 0) {
                            links.push({
                                source: nodeMap.get(`provider-${provider}`),
                                target: nodeMap.get(`type-${type}`),
                                value: value
                            });
                        }
                    });
                });

                // Add type to state links
                resourceTypes.forEach(type => {
                    states.forEach(state => {
                        const key = `${type}->${state}`;
                        const value = flowMap.get(key);
                        if (value > 0) {
                            links.push({
                                source: nodeMap.get(`type-${type}`),
                                target: nodeMap.get(`state-${state}`),
                                value: value
                            });
                        }
                    });
                });
            }
        });

        // If no data, show message
        if (nodes.length === 0 || links.length === 0) {
            const g = svg.append('g')
                .attr('transform', `translate(${margin.left},${margin.top})`);
            
            g.append('text')
                .attr('x', width / 2)
                .attr('y', height / 2)
                .attr('text-anchor', 'middle')
                .text('No resource flow data available')
                .style('font-size', '16px')
                .style('fill', '#666');
            return;
        }

        // Create Sankey generator
        const sankey = d3.sankey()
            .nodeId(d => d.id)
            .nodeWidth(20)
            .nodePadding(10)
            .size([width, height]);

        // Generate the Sankey diagram
        const graph = sankey({
            nodes: nodes.map(d => Object.assign({}, d)),
            links: links.map(d => Object.assign({}, d))
        });

        const g = svg.append('g')
            .attr('transform', `translate(${margin.left},${margin.top})`);

        // Add links
        const link = g.append('g')
            .selectAll('.link')
            .data(graph.links)
            .enter().append('path')
            .attr('class', 'link')
            .attr('d', d3.sankeyLinkHorizontal())
            .style('stroke', '#cbd5e1')
            .style('stroke-width', d => Math.max(1, d.width))
            .style('fill', 'none')
            .style('opacity', 0.5)
            .on('mouseover', function(event, d) {
                d3.select(this).style('opacity', 0.8);
                // Show tooltip
                const tooltip = d3.select('body').append('div')
                    .attr('class', 'tooltip')
                    .style('position', 'absolute')
                    .style('padding', '10px')
                    .style('background', 'rgba(0,0,0,0.8)')
                    .style('color', 'white')
                    .style('border-radius', '4px')
                    .style('pointer-events', 'none')
                    .style('opacity', 0);
                
                tooltip.transition().duration(200).style('opacity', 1);
                tooltip.html(`${d.source.name} → ${d.target.name}<br/>Count: ${d.value}`)
                    .style('left', (event.pageX + 10) + 'px')
                    .style('top', (event.pageY - 28) + 'px');
            })
            .on('mouseout', function() {
                d3.select(this).style('opacity', 0.5);
                d3.selectAll('.tooltip').remove();
            });

        // Add nodes
        const node = g.append('g')
            .selectAll('.node')
            .data(graph.nodes)
            .enter().append('g')
            .attr('class', 'node')
            .attr('transform', d => `translate(${d.x0},${d.y0})`);

        // Add rectangles for nodes
        node.append('rect')
            .attr('height', d => d.y1 - d.y0)
            .attr('width', sankey.nodeWidth())
            .style('fill', d => d.color)
            .style('stroke', '#000')
            .style('stroke-width', 0.5)
            .style('cursor', 'pointer')
            .on('mouseover', function(event, d) {
                d3.select(this).style('opacity', 0.8);
            })
            .on('mouseout', function() {
                d3.select(this).style('opacity', 1);
            });

        // Add labels for nodes
        node.append('text')
            .attr('x', -6)
            .attr('y', d => (d.y1 - d.y0) / 2)
            .attr('dy', '0.35em')
            .attr('text-anchor', 'end')
            .text(d => d.name)
            .style('font-size', '12px')
            .filter(d => d.x0 < width / 2)
            .attr('x', 6 + sankey.nodeWidth())
            .attr('text-anchor', 'start');

        // Add title
        g.append('text')
            .attr('x', width / 2)
            .attr('y', -10)
            .attr('text-anchor', 'middle')
            .text('Resource Flow: Provider → Type → State')
            .style('font-size', '14px')
            .style('font-weight', 'bold');
    }

    // Helper function to get provider color
    getProviderColor(provider) {
        const colors = {
            'aws': '#FF9900',
            'azure': '#0078D4',
            'gcp': '#4285F4',
            'digitalocean': '#0080FF',
            'unknown': '#94a3b8'
        };
        return colors[provider.toLowerCase()] || '#94a3b8';
    }

    // Helper function to get state color
    getStateColor(state) {
        const colors = {
            'active': '#10b981',
            'running': '#10b981',
            'stopped': '#f59e0b',
            'terminated': '#ef4444',
            'deleted': '#ef4444',
            'pending': '#6366f1',
            'unknown': '#94a3b8'
        };
        return colors[state.toLowerCase()] || '#94a3b8';
    }

    // Create Tree Map for State File Hierarchy
    createTreeMap(containerId, stateFiles) {
        const container = document.getElementById(containerId);
        if (!container || !d3) return;

        container.innerHTML = '';

        const width = container.clientWidth;
        const height = 500;

        // Prepare hierarchical data
        const root = {
            name: 'State Files',
            children: stateFiles.map(state => ({
                name: state.name,
                value: state.resource_count || 1,
                health: state.health.status,
                backend: state.type,
                drift: state.drift_percentage || 0,
                id: state.id
            }))
        };

        // Create hierarchy
        const hierarchy = d3.hierarchy(root)
            .sum(d => d.value)
            .sort((a, b) => b.value - a.value);

        // Create treemap layout
        const treemap = d3.treemap()
            .size([width, height])
            .padding(2)
            .round(true);

        treemap(hierarchy);

        // Create SVG
        const svg = d3.select(`#${containerId}`)
            .append('svg')
            .attr('width', width)
            .attr('height', height);

        // Create cells
        const cell = svg.selectAll('g')
            .data(hierarchy.leaves())
            .enter().append('g')
            .attr('transform', d => `translate(${d.x0},${d.y0})`);

        // Add rectangles
        cell.append('rect')
            .attr('width', d => d.x1 - d.x0)
            .attr('height', d => d.y1 - d.y0)
            .attr('fill', d => this.colors[d.data.health] || this.colors.unknown)
            .attr('stroke', '#fff')
            .attr('stroke-width', 2)
            .style('cursor', 'pointer')
            .on('click', (event, d) => {
                this.onStateFileClick(d.data.id);
            });

        // Add text labels for larger cells
        cell.append('text')
            .attr('x', 4)
            .attr('y', 20)
            .text(d => {
                const width = d.x1 - d.x0;
                const height = d.y1 - d.y0;
                if (width > 50 && height > 30) {
                    return d.data.name;
                }
                return '';
            })
            .attr('font-size', '12px')
            .attr('fill', 'white');

        // Add resource count for larger cells
        cell.append('text')
            .attr('x', 4)
            .attr('y', 35)
            .text(d => {
                const width = d.x1 - d.x0;
                const height = d.y1 - d.y0;
                if (width > 50 && height > 45) {
                    return `${d.data.value} resources`;
                }
                return '';
            })
            .attr('font-size', '10px')
            .attr('fill', 'white')
            .attr('opacity', 0.8);

        // Add tooltips
        cell.append('title')
            .text(d => `${d.data.name}\nResources: ${d.data.value}\nHealth: ${d.data.health}\nBackend: ${d.data.backend}\nDrift: ${d.data.drift}%`);
    }

    // Create Perspective Split View
    createPerspectiveSplitView(containerId, perspective) {
        const container = document.getElementById(containerId);
        if (!container) return;

        container.innerHTML = `
            <div class="flex h-full">
                <!-- State's View -->
                <div class="w-1/2 p-4 border-r">
                    <h3 class="text-lg font-bold mb-4">Through State's Eyes</h3>
                    <div class="space-y-2">
                        <div class="stat bg-base-100 shadow rounded-lg p-3">
                            <div class="stat-title text-xs">Managed Resources</div>
                            <div class="stat-value text-2xl">${perspective.statistics.total_managed}</div>
                        </div>
                        <div class="overflow-y-auto max-h-96">
                            ${this.renderManagedResourcesList(perspective.managed_resources)}
                        </div>
                    </div>
                </div>
                
                <!-- Reality View -->
                <div class="w-1/2 p-4">
                    <h3 class="text-lg font-bold mb-4">Reality Check</h3>
                    <div class="space-y-2">
                        <div class="stat bg-base-100 shadow rounded-lg p-3">
                            <div class="stat-title text-xs">Out-of-Band Resources</div>
                            <div class="stat-value text-2xl text-error">${perspective.statistics.total_out_of_band}</div>
                        </div>
                        <div class="overflow-y-auto max-h-96">
                            ${this.renderOutOfBandResourcesList(perspective.out_of_band_resources)}
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    // Render managed resources list
    renderManagedResourcesList(resources) {
        if (!resources || resources.length === 0) {
            return '<p class="text-gray-500">No managed resources</p>';
        }

        return resources.slice(0, 50).map(resource => `
            <div class="p-2 bg-base-200 rounded mb-1 hover:bg-base-300 cursor-pointer">
                <div class="flex justify-between items-center">
                    <span class="text-sm font-medium">${resource.name}</span>
                    <span class="badge badge-sm ${resource.status === 'drifted' ? 'badge-warning' : 'badge-success'}">
                        ${resource.status}
                    </span>
                </div>
                <div class="text-xs text-gray-500">
                    ${resource.type} | ${resource.provider}
                </div>
            </div>
        `).join('');
    }

    // Render out-of-band resources list
    renderOutOfBandResourcesList(resources) {
        if (!resources || resources.length === 0) {
            return '<p class="text-gray-500">No out-of-band resources detected</p>';
        }

        return resources.slice(0, 50).map(resource => `
            <div class="p-2 bg-error bg-opacity-10 rounded mb-1 hover:bg-opacity-20 cursor-pointer">
                <div class="flex justify-between items-center">
                    <span class="text-sm font-medium">${resource.name}</span>
                    <span class="badge badge-sm badge-error">
                        ${resource.adoption_priority}
                    </span>
                </div>
                <div class="text-xs text-gray-500">
                    ${resource.type} | ${resource.reason}
                </div>
                <div class="text-xs mt-1">
                    <code class="text-xs bg-base-300 p-1 rounded">
                        ${resource.suggested_import}
                    </code>
                </div>
            </div>
        `).join('');
    }

    // Create Resource Dependency Graph
    createDependencyGraph(containerId, resourceGraph) {
        const container = document.getElementById(containerId);
        if (!container || !d3 || !resourceGraph) return;

        container.innerHTML = '';

        const width = container.clientWidth;
        const height = 600;

        const svg = d3.select(`#${containerId}`)
            .append('svg')
            .attr('width', width)
            .attr('height', height);

        // Create force simulation for dependency graph
        const simulation = d3.forceSimulation(resourceGraph.nodes)
            .force('link', d3.forceLink(resourceGraph.edges)
                .id(d => d.id)
                .distance(150))
            .force('charge', d3.forceManyBody().strength(-500))
            .force('center', d3.forceCenter(width / 2, height / 2));

        // Add arrow markers for directed edges
        svg.append('defs').selectAll('marker')
            .data(['depends_on', 'references', 'creates'])
            .enter().append('marker')
            .attr('id', d => d)
            .attr('viewBox', '0 -5 10 10')
            .attr('refX', 25)
            .attr('refY', 0)
            .attr('markerWidth', 6)
            .attr('markerHeight', 6)
            .attr('orient', 'auto')
            .append('path')
            .attr('d', 'M0,-5L10,0L0,5')
            .attr('fill', '#999');

        // Add links
        const link = svg.append('g')
            .selectAll('line')
            .data(resourceGraph.edges)
            .enter().append('line')
            .attr('stroke', d => {
                switch(d.type) {
                    case 'depends_on': return '#ff6b6b';
                    case 'references': return '#4ecdc4';
                    case 'creates': return '#95e77e';
                    default: return '#999';
                }
            })
            .attr('stroke-width', 2)
            .attr('marker-end', d => `url(#${d.type})`);

        // Add nodes
        const node = svg.append('g')
            .selectAll('g')
            .data(resourceGraph.nodes)
            .enter().append('g')
            .call(d3.drag()
                .on('start', dragstarted)
                .on('drag', dragged)
                .on('end', dragended));

        // Add circles for nodes
        node.append('circle')
            .attr('r', 20)
            .attr('fill', d => {
                if (d.status === 'missing') return this.colors.critical;
                if (d.status === 'drifted') return this.colors.warning;
                return this.colors.managed;
            })
            .attr('stroke', '#fff')
            .attr('stroke-width', 2);

        // Add icons or text
        node.append('text')
            .text(d => d.label.substring(0, 2).toUpperCase())
            .attr('x', 0)
            .attr('y', 5)
            .attr('text-anchor', 'middle')
            .attr('fill', 'white')
            .attr('font-size', '12px')
            .attr('font-weight', 'bold');

        // Add labels
        node.append('text')
            .text(d => d.label)
            .attr('x', 0)
            .attr('y', 35)
            .attr('text-anchor', 'middle')
            .attr('font-size', '11px');

        // Add tooltips
        node.append('title')
            .text(d => `${d.label}\nType: ${d.type}\nProvider: ${d.provider}\nStatus: ${d.status}`);

        // Update positions on tick
        simulation.on('tick', () => {
            link
                .attr('x1', d => d.source.x)
                .attr('y1', d => d.source.y)
                .attr('x2', d => d.target.x)
                .attr('y2', d => d.target.y);

            node.attr('transform', d => `translate(${d.x},${d.y})`);
        });

        // Drag functions
        function dragstarted(event, d) {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            d.fx = d.x;
            d.fy = d.y;
        }

        function dragged(event, d) {
            d.fx = event.x;
            d.fy = event.y;
        }

        function dragended(event, d) {
            if (!event.active) simulation.alphaTarget(0);
            d.fx = null;
            d.fy = null;
        }
    }

    // Create Timeline Visualization
    createTimeline(containerId, events) {
        const container = document.getElementById(containerId);
        if (!container || !d3) return;

        container.innerHTML = '';

        const width = container.clientWidth;
        const height = 200;
        const margin = { top: 20, right: 20, bottom: 40, left: 50 };

        const svg = d3.select(`#${containerId}`)
            .append('svg')
            .attr('width', width)
            .attr('height', height);

        if (!events || events.length === 0) {
            svg.append('text')
                .attr('x', width / 2)
                .attr('y', height / 2)
                .attr('text-anchor', 'middle')
                .text('No timeline data available')
                .style('fill', '#999');
            return;
        }

        // Parse dates and sort events
        events.forEach(d => {
            d.date = new Date(d.timestamp);
        });
        events.sort((a, b) => a.date - b.date);

        // Create scales
        const xScale = d3.scaleTime()
            .domain(d3.extent(events, d => d.date))
            .range([margin.left, width - margin.right]);

        const yScale = d3.scaleOrdinal()
            .domain(['apply', 'refresh', 'drift', 'import'])
            .range([40, 60, 80, 100]);

        // Add x-axis
        svg.append('g')
            .attr('transform', `translate(0,${height - margin.bottom})`)
            .call(d3.axisBottom(xScale)
                .tickFormat(d3.timeFormat('%b %d')));

        // Add events
        const eventGroup = svg.append('g');

        eventGroup.selectAll('circle')
            .data(events)
            .enter().append('circle')
            .attr('cx', d => xScale(d.date))
            .attr('cy', d => yScale(d.event_type))
            .attr('r', 6)
            .attr('fill', d => {
                switch(d.event_type) {
                    case 'apply': return this.colors.fresh;
                    case 'refresh': return this.colors.managed;
                    case 'drift': return this.colors.warning;
                    case 'import': return this.colors.recent;
                    default: return this.colors.unknown;
                }
            })
            .attr('stroke', '#fff')
            .attr('stroke-width', 2);

        // Add event labels
        eventGroup.selectAll('text')
            .data(events)
            .enter().append('text')
            .attr('x', d => xScale(d.date))
            .attr('y', d => yScale(d.event_type) - 10)
            .attr('text-anchor', 'middle')
            .attr('font-size', '10px')
            .text(d => d.description);

        // Add tooltips
        eventGroup.selectAll('circle')
            .append('title')
            .text(d => `${d.event_type}: ${d.description}\n${d.date.toLocaleString()}`);
    }

    // Create Coverage Meter
    createCoverageMeter(containerId, coverage) {
        const container = document.getElementById(containerId);
        if (!container) return;

        const percentage = Math.round(coverage);
        const color = percentage >= 80 ? this.colors.fresh :
                     percentage >= 60 ? this.colors.recent :
                     percentage >= 40 ? this.colors.stale :
                     this.colors.critical;

        container.innerHTML = `
            <div class="relative w-48 h-48 mx-auto">
                <svg class="w-full h-full transform -rotate-90">
                    <circle
                        cx="96"
                        cy="96"
                        r="88"
                        stroke="#e5e7eb"
                        stroke-width="8"
                        fill="none"
                    />
                    <circle
                        cx="96"
                        cy="96"
                        r="88"
                        stroke="${color}"
                        stroke-width="8"
                        fill="none"
                        stroke-dasharray="${2 * Math.PI * 88}"
                        stroke-dashoffset="${2 * Math.PI * 88 * (1 - percentage / 100)}"
                        class="transition-all duration-1000"
                    />
                </svg>
                <div class="absolute inset-0 flex flex-col items-center justify-center">
                    <span class="text-4xl font-bold">${percentage}%</span>
                    <span class="text-sm text-gray-500">Coverage</span>
                </div>
            </div>
        `;
    }

    // Handle state file click
    onStateFileClick(stateFileId) {
        // Emit custom event
        window.dispatchEvent(new CustomEvent('stateFileSelected', {
            detail: { stateFileId }
        }));
    }

    // Clean up visualizations
    destroy() {
        // Clean up any active D3 simulations or charts
        Object.values(this.graphInstances).forEach(instance => {
            if (instance && instance.stop) {
                instance.stop();
            }
        });
        this.graphInstances = {};
        this.charts = {};
    }
}

// Export for use
window.StateVisualizations = StateVisualizations;