// Main app.js for the GavelEngine Business Rules Management UI

document.addEventListener('DOMContentLoaded', function() {
    // Initialize the application
    initApp();
});

// Global variables
let currentEngine = null;
let eventParamsEditor = null;
let conditionsJsonEditor = null;
let runtimeFactsEditor = null;
let availableFacts = [];
let darkMode = localStorage.getItem('darkMode') === 'true';

// Initialize application
function initApp() {
    // Initialize dark mode if enabled
    if (darkMode) {
        document.body.classList.add('dark-mode');
    }
    
    // Add theme toggle button to header
    const headerRight = document.querySelector('.header-right');
    const themeToggle = document.createElement('button');
    themeToggle.className = 'theme-toggle';
    themeToggle.innerHTML = darkMode ? 
        '<i class="fas fa-sun"></i>' : 
        '<i class="fas fa-moon"></i>';
    themeToggle.addEventListener('click', toggleDarkMode);
    headerRight.insertBefore(themeToggle, headerRight.firstChild);
    
    // Initialize CodeMirror editors
    initCodeMirrorEditors();
    
    // Setup navigation
    setupNavigation();
    
    // Setup engine selector
    setupEngineSelector();
    
    // Setup form submissions
    setupFormSubmissions();
    
    // Setup condition builder
    setupConditionBuilder();
    
    // Load engines on startup
    fetchEngines();
    
    // Load predefined facts
    fetchPredefinedFacts();
}

// Toggle dark mode
function toggleDarkMode() {
    darkMode = !darkMode;
    
    if (darkMode) {
        document.body.classList.add('dark-mode');
        document.querySelector('.theme-toggle').innerHTML = '<i class="fas fa-sun"></i>';
    } else {
        document.body.classList.remove('dark-mode');
        document.querySelector('.theme-toggle').innerHTML = '<i class="fas fa-moon"></i>';
    }
    
    localStorage.setItem('darkMode', darkMode);
    
    // Update CodeMirror theme
    const theme = darkMode ? 'dracula' : 'default';
    eventParamsEditor.setOption('theme', theme);
    conditionsJsonEditor.setOption('theme', theme);
    runtimeFactsEditor.setOption('theme', theme);
}

// Initialize CodeMirror editors
function initCodeMirrorEditors() {
    const theme = darkMode ? 'dracula' : 'default';
    
    // Event params editor
    eventParamsEditor = CodeMirror(document.getElementById('event-params-editor'), {
        mode: { name: 'javascript', json: true },
        theme: theme,
        lineNumbers: true,
        matchBrackets: true,
        autoCloseBrackets: true,
        tabSize: 2,
        value: '{\n  "message": "Event triggered"\n}'
    });
    
    // Conditions JSON editor
    conditionsJsonEditor = CodeMirror(document.getElementById('conditions-json-editor'), {
        mode: { name: 'javascript', json: true },
        theme: theme,
        lineNumbers: true,
        matchBrackets: true,
        autoCloseBrackets: true,
        tabSize: 2,
        value: '{\n  "all": [\n    {\n      "fact": "age",\n      "operator": "greaterThanInclusive",\n      "value": 18\n    }\n  ]\n}'
    });
    
    // Runtime facts editor
    runtimeFactsEditor = CodeMirror(document.getElementById('runtime-facts-editor'), {
        mode: { name: 'javascript', json: true },
        theme: theme,
        lineNumbers: true,
        matchBrackets: true,
        autoCloseBrackets: true,
        tabSize: 2,
        value: '{\n  "age": 25,\n  "score": 85\n}'
    });
}

// Setup navigation
function setupNavigation() {
    // Handle sidebar link clicks
    document.querySelectorAll('.sidebar-link').forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const pageId = this.getAttribute('data-page');
            
            // Update active link
            document.querySelectorAll('.sidebar-link').forEach(el => el.classList.remove('active'));
            this.classList.add('active');
            
            // Show page
            document.querySelectorAll('.content-page').forEach(page => page.classList.remove('active'));
            document.getElementById(pageId + '-page').classList.add('active');
            
            // Update page title
            document.getElementById('page-title').textContent = this.querySelector('span').textContent;
        });
    });
    
    // Handle quick action buttons
    document.querySelectorAll('.quick-action-btn').forEach(btn => {
        btn.addEventListener('click', function(e) {
            e.preventDefault();
            const pageId = this.getAttribute('data-page');
            
            // Click corresponding sidebar link
            document.querySelector(`.sidebar-link[data-page="${pageId}"]`).click();
        });
    });
    
    // Handle sidebar toggle
    document.getElementById('sidebar-toggle').addEventListener('click', function() {
        const sidebar = document.querySelector('.sidebar');
        const mainContent = document.querySelector('.main-content');
        
        sidebar.classList.toggle('expanded');
        mainContent.classList.toggle('sidebar-expanded');
    });
}

// Setup engine selector
function setupEngineSelector() {
    const engineSelector = document.getElementById('engine-selector');
    
    engineSelector.addEventListener('change', function() {
        const engineName = this.value;
        if (engineName) {
            currentEngine = engineName;
            document.getElementById('current-engine').textContent = `Engine: ${engineName}`;
            
            // Update UI to show engine-specific content
            document.querySelectorAll('.engine-required-alert').forEach(alert => alert.classList.add('d-none'));
            document.querySelectorAll('.engine-content').forEach(content => content.classList.remove('d-none'));
            
            // Fetch engine data
            fetchFacts();
            fetchRules();
        } else {
            currentEngine = null;
            document.getElementById('current-engine').textContent = 'No engine selected';
            
            // Hide engine-specific content
            document.querySelectorAll('.engine-required-alert').forEach(alert => alert.classList.remove('d-none'));
            document.querySelectorAll('.engine-content').forEach(content => content.classList.add('d-none'));
        }
    });
}

// Setup form submissions
function setupFormSubmissions() {
    // Create engine form
    const createEngineForm = document.getElementById('create-engine-form');
    if (createEngineForm) {
        createEngineForm.addEventListener('submit', function(e) {
            e.preventDefault();
            const engineName = document.getElementById('engine-name').value.trim();
            
            createEngine(engineName);
        });
    }
    
    // Add fact form
    const addFactForm = document.getElementById('add-fact-form');
    if (addFactForm) {
        // Toggle fact value input based on type
        document.getElementById('fact-type').addEventListener('change', function() {
            const factType = this.value;
            if (factType === 'constant') {
                document.getElementById('constant-value-group').classList.remove('d-none');
                document.getElementById('function-value-group').classList.add('d-none');
            } else {
                document.getElementById('constant-value-group').classList.add('d-none');
                document.getElementById('function-value-group').classList.remove('d-none');
            }
        });
        
        addFactForm.addEventListener('submit', function(e) {
            e.preventDefault();
            if (!currentEngine) {
                showToast('Please select an engine first', 'error');
                return;
            }
            
            const factId = document.getElementById('fact-id').value.trim();
            const factType = document.getElementById('fact-type').value;
            let factValue = null;
            let description = '';
            
            if (factType === 'constant') {
                const rawValue = document.getElementById('constant-value').value.trim();
                // Try to convert to appropriate type
                if (rawValue === 'true') {
                    factValue = true;
                } else if (rawValue === 'false') {
                    factValue = false;
                } else if (!isNaN(rawValue) && rawValue !== '') {
                    factValue = Number(rawValue);
                } else {
                    factValue = rawValue;
                }
            } else {
                description = document.getElementById('function-description').value.trim();
            }
            
            const cache = document.getElementById('fact-cache').checked;
            
            addFact(factId, factType, factValue, description, cache);
        });
    }
    
    // Create rule form
    const createRuleForm = document.getElementById('create-rule-form');
    if (createRuleForm) {
        createRuleForm.addEventListener('submit', function(e) {
            e.preventDefault();
            if (!currentEngine) {
                showToast('Please select an engine first', 'error');
                return;
            }
            
            const ruleName = document.getElementById('rule-name').value.trim();
            const priority = parseInt(document.getElementById('rule-priority').value) || 1;
            const eventType = document.getElementById('event-type').value.trim();
            
            // Get event params
            let eventParams = {};
            try {
                eventParams = JSON.parse(eventParamsEditor.getValue());
            } catch (error) {
                showToast('Invalid event parameters JSON: ' + error.message, 'error');
                return;
            }
            
            // Get conditions
            let conditions = {};
            const activeTab = document.querySelector('.nav-link.active').getAttribute('id');
            
            if (activeTab === 'visual-tab') {
                conditions = buildConditionsFromVisualBuilder();
            } else {
                try {
                    conditions = JSON.parse(conditionsJsonEditor.getValue());
                } catch (error) {
                    showToast('Invalid conditions JSON: ' + error.message, 'error');
                    return;
                }
            }
            
            const event = {
                type: eventType,
                params: eventParams
            };
            
            addRule(ruleName, priority, conditions, event);
        });
    }
    
    // Run engine form
    const runEngineForm = document.getElementById('run-engine-form');
    if (runEngineForm) {
        runEngineForm.addEventListener('submit', function(e) {
            e.preventDefault();
            if (!currentEngine) {
                showToast('Please select an engine first', 'error');
                return;
            }
            
            let runtimeFacts = {};
            try {
                runtimeFacts = JSON.parse(runtimeFactsEditor.getValue());
            } catch (error) {
                showToast('Invalid runtime facts JSON: ' + error.message, 'error');
                return;
            }
            
            runEngine(runtimeFacts);
        });
    }
}

// Setup condition builder
function setupConditionBuilder() {
    // Add condition button
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('add-condition') || e.target.closest('.add-condition')) {
            const groupContainer = e.target.closest('.condition-group').querySelector('.condition-items');
            addConditionItem(groupContainer);
        }
    });
    
    // Remove condition button
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('remove-condition') || e.target.closest('.remove-condition')) {
            const item = e.target.closest('.condition-item');
            item.remove();
        }
    });
    
    // Add nested condition group
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('add-nested-condition') || e.target.closest('.add-nested-condition')) {
            const item = e.target.closest('.condition-item');
            addNestedConditionGroup(item);
        }
    });
    
    // Change condition group type (AND/OR)
    document.addEventListener('change', function(e) {
        if (e.target.classList.contains('condition-group-type')) {
            // The visual representation stays the same,
            // but this will affect how we build the conditions object
        }
    });
    
    // Sync visual builder with JSON editor
    document.querySelectorAll('.nav-link').forEach(tab => {
        tab.addEventListener('click', function() {
            if (this.id === 'json-tab') {
                // Update JSON editor with current visual builder state
                const conditions = buildConditionsFromVisualBuilder();
                conditionsJsonEditor.setValue(JSON.stringify(conditions, null, 2));
            } else if (this.id === 'visual-tab') {
                try {
                    // Update visual builder with JSON editor state
                    const conditions = JSON.parse(conditionsJsonEditor.getValue());
                    updateVisualBuilderFromConditions(conditions);
                } catch (error) {
                    showToast('Invalid JSON: ' + error.message, 'error');
                }
            }
        });
    });
}

// API functions

// Fetch engines
function fetchEngines() {
    fetch('/api/engines')
        .then(response => response.json())
        .then(data => {
            // Update engine selector
            const engineSelector = document.getElementById('engine-selector');
            engineSelector.innerHTML = '<option value="">Select Engine</option>';
            
            if (data.engines && data.engines.length > 0) {
                data.engines.forEach(engine => {
                    const option = document.createElement('option');
                    option.value = engine;
                    option.textContent = engine;
                    engineSelector.appendChild(option);
                });
                
                // Update stat
                document.getElementById('stat-engines').textContent = data.engines.length;
            } else {
                document.getElementById('stat-engines').textContent = '0';
            }
            
            // Update engines list
            const enginesList = document.getElementById('engines-list');
            if (enginesList) {
                if (data.engines && data.engines.length > 0) {
                    let html = '';
                    data.engines.forEach(engine => {
                        html += `
                            <tr data-engine="${engine}">
                                <td>${engine}</td>
                                <td>0</td>
                                <td>0</td>
                                <td>
                                    <button class="btn btn-sm btn-danger delete-engine" data-engine="${engine}">
                                        <i class="fas fa-trash"></i>
                                    </button>
                                </td>
                            </tr>
                        `;
                    });
                    enginesList.innerHTML = html;
                    
                    // Add click handler for engine rows
                    enginesList.querySelectorAll('tr[data-engine]').forEach(row => {
                        row.addEventListener('click', function(e) {
                            if (!e.target.closest('.btn')) {
                                const engine = this.getAttribute('data-engine');
                                document.getElementById('engine-selector').value = engine;
                                document.getElementById('engine-selector').dispatchEvent(new Event('change'));
                            }
                        });
                    });
                    
                    // Add delete engine handler
                    enginesList.querySelectorAll('.delete-engine').forEach(btn => {
                        btn.addEventListener('click', function() {
                            const engine = this.getAttribute('data-engine');
                            if (confirm(`Are you sure you want to delete engine "${engine}"?`)) {
                                deleteEngine(engine);
                            }
                        });
                    });
                } else {
                    enginesList.innerHTML = '<tr><td colspan="4" class="text-center">No engines found.</td></tr>';
                }
            }
        })
        .catch(error => {
            showToast('Error fetching engines: ' + error.message, 'error');
        });
}

// Create engine
function createEngine(name) {
    fetch('/api/engines', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ name })
    })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to create engine');
            }
            return response.json();
        })
        .then(data => {
            showToast(`Engine "${name}" created successfully`, 'success');
            document.getElementById('engine-name').value = '';
            
            // Refresh engines
            fetchEngines();
            
            // Select the new engine
            setTimeout(() => {
                document.getElementById('engine-selector').value = name;
                document.getElementById('engine-selector').dispatchEvent(new Event('change'));
            }, 300);
        })
        .catch(error => {
            showToast('Error creating engine: ' + error.message, 'error');
        });
}

// Delete engine
function deleteEngine(name) {
    fetch(`/api/engines/${name}`, {
        method: 'DELETE'
    })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to delete engine');
            }
            return response.json();
        })
        .then(data => {
            showToast(`Engine "${name}" deleted successfully`, 'success');
            
            // If current engine was deleted, reset selector
            if (currentEngine === name) {
                currentEngine = null;
                document.getElementById('current-engine').textContent = 'No engine selected';
                document.getElementById('engine-selector').value = '';
                
                // Hide engine-specific content
                document.querySelectorAll('.engine-required-alert').forEach(alert => alert.classList.remove('d-none'));
                document.querySelectorAll('.engine-content').forEach(content => content.classList.add('d-none'));
            }
            
            // Refresh engines
            fetchEngines();
        })
        .catch(error => {
            showToast('Error deleting engine: ' + error.message, 'error');
        });
}

// Fetch facts
function fetchFacts() {
    if (!currentEngine) {
        return;
    }
    
    fetch(`/api/engines/${currentEngine}/facts`)
        .then(response => response.json())
        .then(data => {
            availableFacts = data.facts || [];
            
            // Update stat
            document.getElementById('stat-facts').textContent = availableFacts.length;
            
            // Update facts list
            const factsList = document.getElementById('facts-list');
            if (factsList) {
                if (availableFacts.length > 0) {
                    let html = '';
                    availableFacts.forEach(fact => {
                        html += `
                            <tr>
                                <td>${fact.id}</td>
                                <td>${fact.isConstant ? 'Constant' : 'Function'}</td>
                                <td>${fact.cache ? '<i class="fas fa-check text-success"></i>' : '<i class="fas fa-times text-danger"></i>'}</td>
                                <td>${fact.description || '-'}</td>
                                <td>
                                    <button class="btn btn-sm btn-danger delete-fact" data-fact="${fact.id}">
                                        <i class="fas fa-trash"></i>
                                    </button>
                                </td>
                            </tr>
                        `;
                    });
                    factsList.innerHTML = html;
                    
                    // Add delete fact handler
                    factsList.querySelectorAll('.delete-fact').forEach(btn => {
                        btn.addEventListener('click', function() {
                            const factId = this.getAttribute('data-fact');
                            if (confirm(`Are you sure you want to delete fact "${factId}"?`)) {
                                deleteFact(factId);
                            }
                        });
                    });
                } else {
                    factsList.innerHTML = '<tr><td colspan="5" class="text-center">No facts found.</td></tr>';
                }
            }
            
            // Update fact selectors in condition builder
            document.querySelectorAll('.fact-select').forEach(select => {
                const currentValue = select.value;
                select.innerHTML = '<option value="" disabled selected>Select Fact</option>';
                
                availableFacts.forEach(fact => {
                    const option = document.createElement('option');
                    option.value = fact.id;
                    option.textContent = fact.id;
                    select.appendChild(option);
                });
                
                if (currentValue && select.querySelector(`option[value="${currentValue}"]`)) {
                    select.value = currentValue;
                }
            });
        })
        .catch(error => {
            showToast('Error fetching facts: ' + error.message, 'error');
        });
}

// Add fact
function addFact(id, type, value, description, cache) {
    if (!currentEngine) {
        showToast('Please select an engine first', 'error');
        return;
    }
    
    const factData = {
        id,
        type,
        cache
    };
    
    if (type === 'constant') {
        factData.value = value;
    } else {
        factData.description = description;
    }
    
    fetch(`/api/engines/${currentEngine}/facts`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(factData)
    })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to add fact');
            }
            return response.json();
        })
        .then(data => {
            showToast(`Fact "${id}" added successfully`, 'success');
            
            // Reset form
            document.getElementById('fact-id').value = '';
            document.getElementById('fact-type').value = 'constant';
            document.getElementById('constant-value').value = '';
            document.getElementById('function-description').value = '';
            document.getElementById('fact-cache').checked = true;
            document.getElementById('constant-value-group').classList.remove('d-none');
            document.getElementById('function-value-group').classList.add('d-none');
            
            // Refresh facts
            fetchFacts();
        })
        .catch(error => {
            showToast('Error adding fact: ' + error.message, 'error');
        });
}

// Delete fact
function deleteFact(id) {
    if (!currentEngine) {
        showToast('Please select an engine first', 'error');
        return;
    }
    
    fetch(`/api/engines/${currentEngine}/facts/${id}`, {
        method: 'DELETE'
    })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to delete fact');
            }
            return response.json();
        })
        .then(data => {
            showToast(`Fact "${id}" deleted successfully`, 'success');
            
            // Refresh facts
            fetchFacts();
        })
        .catch(error => {
            showToast('Error deleting fact: ' + error.message, 'error');
        });
}

// Fetch predefined facts
function fetchPredefinedFacts() {
    fetch('/api/predefined-facts')
        .then(response => response.json())
        .then(data => {
            const predefinedFactsList = document.getElementById('predefined-facts-list');
            if (predefinedFactsList) {
                if (data.facts && data.facts.length > 0) {
                    let html = '';
                    data.facts.forEach(fact => {
                        html += `
                            <tr>
                                <td>${fact.id}</td>
                                <td>${fact.description}</td>
                                <td>
                                    <button class="btn btn-sm btn-primary add-predefined-fact" data-fact-id="${fact.id}" data-fact-description="${fact.description}">
                                        <i class="fas fa-plus"></i> Add
                                    </button>
                                </td>
                            </tr>
                        `;
                    });
                    predefinedFactsList.innerHTML = html;
                    
                    // Add predefined fact handler
                    predefinedFactsList.querySelectorAll('.add-predefined-fact').forEach(btn => {
                        btn.addEventListener('click', function() {
                            if (!currentEngine) {
                                showToast('Please select an engine first', 'error');
                                return;
                            }
                            
                            const factId = this.getAttribute('data-fact-id');
                            const description = this.getAttribute('data-fact-description');
                            
                            // Set values in the form
                            document.getElementById('fact-id').value = factId;
                            document.getElementById('fact-type').value = 'function';
                            document.getElementById('function-description').value = description;
                            
                            // Trigger fact type change
                            document.getElementById('fact-type').dispatchEvent(new Event('change'));
                            
                            // Scroll to form
                            document.getElementById('add-fact-form').scrollIntoView({ behavior: 'smooth' });
                        });
                    });
                } else {
                    predefinedFactsList.innerHTML = '<tr><td colspan="3" class="text-center">No predefined facts available.</td></tr>';
                }
            }
        })
        .catch(error => {
            showToast('Error fetching predefined facts: ' + error.message, 'error');
        });
}

// Fetch rules
function fetchRules() {
    if (!currentEngine) {
        return;
    }
    
    fetch(`/api/engines/${currentEngine}/rules`)
        .then(response => response.json())
        .then(data => {
            console.log("Raw rules data received:", data);
            
            let rules = [];
            
            // Handle different possible data formats
            if (data.rules && Array.isArray(data.rules)) {
                // Standard array format from our updated server
                rules = data.rules;
            } else if (typeof data === 'object' && !Array.isArray(data) && !data.rules) {
                // Handle object format with rule names as keys
                rules = Object.keys(data).map(key => {
                    const rule = data[key];
                    // Ensure event property is properly formatted
                    if (rule.event && (rule.event.Type || rule.event.Params)) {
                        return {
                            name: rule.name || key,
                            priority: rule.priority || 1,
                            event: {
                                type: rule.event.Type?.toLowerCase() || rule.event.type || 'unknown',
                                params: rule.event.Params || rule.event.params || {}
                            }
                        };
                    }
                    return rule;
                });
            }
            
            console.log("Processed rules:", rules);
            
            // Update stat
            document.getElementById('stat-rules').textContent = rules.length;
            
            // Update rules list
            const rulesList = document.getElementById('rules-list');
            
            if (rulesList) {
                if (rules.length > 0) {
                    let html = '';
                    rules.forEach(rule => {
                        // Ensure type exists and is lowercase
                        const eventType = rule.event?.type || rule.event?.Type || 'unknown';
                        
                        html += `
                            <tr>
                                <td>${rule.name}</td>
                                <td>${rule.priority}</td>
                                <td>${eventType}</td>
                                <td>
                                    <button class="btn btn-sm btn-info view-rule" data-rule="${rule.name}">
                                        <i class="fas fa-eye"></i>
                                    </button>
                                    <button class="btn btn-sm btn-danger delete-rule" data-rule="${rule.name}">
                                        <i class="fas fa-trash"></i>
                                    </button>
                                </td>
                            </tr>
                        `;
                    });
                    
                    rulesList.innerHTML = html;
                    
                    // Make sure the engine content is visible
                    document.querySelectorAll('.engine-required-alert').forEach(alert => alert.classList.add('d-none'));
                    document.querySelectorAll('.engine-content').forEach(content => content.classList.remove('d-none'));
                    
                    // Add view rule handler
                    rulesList.querySelectorAll('.view-rule').forEach(btn => {
                        btn.addEventListener('click', function() {
                            const ruleName = this.getAttribute('data-rule');
                            viewRule(ruleName);
                        });
                    });
                    
                    // Add delete rule handler
                    rulesList.querySelectorAll('.delete-rule').forEach(btn => {
                        btn.addEventListener('click', function() {
                            const ruleName = this.getAttribute('data-rule');
                            if (confirm(`Are you sure you want to delete rule "${ruleName}"?`)) {
                                deleteRule(ruleName);
                            }
                        });
                    });
                } else {
                    rulesList.innerHTML = '<tr><td colspan="4" class="text-center">No rules found.</td></tr>';
                }
            } else {
                console.error("Could not find rules-list element in the DOM");
            }
        })
        .catch(error => {
            console.error("Error fetching rules:", error);
            showToast('Error fetching rules: ' + error.message, 'error');
        });
}
// View rule details
function viewRule(ruleName) {
    if (!currentEngine) {
        showToast('Please select an engine first', 'error');
        return;
    }
    
    fetch(`/api/engines/${currentEngine}/rules/${ruleName}`)
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to fetch rule details');
            }
            return response.json();
        })
        .then(data => {
            const rule = data.rule;
            
            // Set modal content
            document.getElementById('rule-conditions-preview').textContent = JSON.stringify(rule.conditions, null, 2);
            document.getElementById('rule-event-preview').textContent = JSON.stringify(rule.event, null, 2);
            
            // Show modal
            const modal = new bootstrap.Modal(document.getElementById('rule-details-modal'));
            modal.show();
        })
        .catch(error => {
            showToast('Error fetching rule details: ' + error.message, 'error');
        });
}

// Add rule
function addRule(name, priority, conditions, event) {
    if (!currentEngine) {
        showToast('Please select an engine first', 'error');
        return;
    }
    
    const ruleData = {
        name,
        priority,
        conditions,
        event
    };
    
    fetch(`/api/engines/${currentEngine}/rules`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(ruleData)
    })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to add rule');
            }
            return response.json();
        })
        .then(data => {
            showToast(`Rule "${name}" added successfully`, 'success');
            
            // Reset form
            document.getElementById('rule-name').value = '';
            document.getElementById('rule-priority').value = '1';
            document.getElementById('event-type').value = '';
            eventParamsEditor.setValue('{\n  "message": "Event triggered"\n}');
            
            // Reset condition builder
            document.getElementById('condition-builder').innerHTML = `
                <div class="condition-group">
                    <div class="condition-group-header">
                        <select class="form-select condition-group-type">
                            <option value="all">ALL of these conditions (AND)</option>
                            <option value="any">ANY of these conditions (OR)</option>
                        </select>
                        <button type="button" class="btn btn-sm btn-primary add-condition">
                            <i class="fas fa-plus"></i> Add Condition
                        </button>
                    </div>
                    <div class="condition-items">
                        <!-- Conditions will be added here -->
                    </div>
                </div>
            `;
            
            // Reset JSON editor
            conditionsJsonEditor.setValue('{\n  "all": [\n    {\n      "fact": "age",\n      "operator": "greaterThanInclusive",\n      "value": 18\n    }\n  ]\n}');
            
            // Refresh rules
            fetchRules();
        })
        .catch(error => {
            showToast('Error adding rule: ' + error.message, 'error');
        });
}

// Delete rule
function deleteRule(name) {
    if (!currentEngine) {
        showToast('Please select an engine first', 'error');
        return;
    }
    
    fetch(`/api/engines/${currentEngine}/rules/${name}`, {
        method: 'DELETE'
    })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to delete rule');
            }
            return response.json();
        })
        .then(data => {
            showToast(`Rule "${name}" deleted successfully`, 'success');
            
            // Refresh rules
            fetchRules();
        })
        .catch(error => {
            showToast('Error deleting rule: ' + error.message, 'error');
        });
}

// Run engine
function runEngine(runtimeFacts) {
    if (!currentEngine) {
        showToast('Please select an engine first', 'error');
        return;
    }
    
    fetch(`/api/engines/${currentEngine}/run`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(runtimeFacts)
    })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to run engine');
            }
            return response.json();
        })
        .then(data => {
            showToast('Engine run successfully', 'success');
            
            // Hide placeholder
            document.getElementById('run-results-placeholder').classList.add('d-none');
            
            // Show results
            document.getElementById('run-results').classList.remove('d-none');
            
            // Update events list
            const eventsList = document.getElementById('events-list');
            if (data.events && data.events.length > 0) {
                let html = '';
                data.events.forEach(event => {
                    html += `
                        <div class="result-item">
                            <h5 class="mb-2">${event.type}</h5>
                            <pre class="mb-0">${JSON.stringify(event.params, null, 2)}</pre>
                        </div>
                    `;
                });
                eventsList.innerHTML = html;
            } else {
                eventsList.innerHTML = '<div class="alert alert-info">No events triggered.</div>';
            }
            
            // Update rule results list
            const ruleResultsList = document.getElementById('rule-results-list');
            if (data.ruleResults && data.ruleResults.length > 0) {
                let html = '';
                data.ruleResults.forEach(result => {
                    html += `
                        <div class="result-item">
                            <h5 class="mb-2">${result.name}</h5>
                            <div class="badge bg-${result.success ? 'success' : 'danger'} mb-0">
                                ${result.success ? 'Success' : 'Failure'}
                            </div>
                        </div>
                    `;
                });
                ruleResultsList.innerHTML = html;
            } else {
                ruleResultsList.innerHTML = '<div class="alert alert-info">No rule results available.</div>';
            }
            
            // Update raw JSON
            document.getElementById('raw-results').textContent = JSON.stringify(data, null, 2);
        })
        .catch(error => {
            showToast('Error running engine: ' + error.message, 'error');
        });
}

// Helper functions

// Add condition item to container
function addConditionItem(container) {
    const template = document.getElementById('condition-item-template');
    const clone = document.importNode(template.content, true);
    
    // Populate fact select with available facts
    const factSelect = clone.querySelector('.fact-select');
    factSelect.innerHTML = '<option value="" disabled selected>Select Fact</option>';
    
    availableFacts.forEach(fact => {
        const option = document.createElement('option');
        option.value = fact.id;
        option.textContent = fact.id;
        factSelect.appendChild(option);
    });
    
    container.appendChild(clone);
}

// Add nested condition group
function addNestedConditionGroup(afterElement) {
    const html = `
        <div class="condition-group mt-3">
            <div class="condition-group-header">
                <select class="form-select condition-group-type">
                    <option value="all">ALL of these conditions (AND)</option>
                    <option value="any">ANY of these conditions (OR)</option>
                </select>
                <button type="button" class="btn btn-sm btn-primary add-condition">
                    <i class="fas fa-plus"></i> Add Condition
                </button>
            </div>
            <div class="condition-items">
                <!-- Conditions will be added here -->
            </div>
        </div>
    `;
    
    // Create temporary element to hold the HTML
    const temp = document.createElement('div');
    temp.innerHTML = html;
    
    // Insert after the provided element
    afterElement.after(temp.firstElementChild);
}

// Build conditions object from visual builder
function buildConditionsFromVisualBuilder() {
    const rootGroup = document.querySelector('#condition-builder > .condition-group');
    return buildConditionGroupObject(rootGroup);
}

// Recursively build condition group object
function buildConditionGroupObject(groupElement) {
    const type = groupElement.querySelector('.condition-group-type').value;
    const result = {};
    result[type] = [];
    
    // Process direct condition items
    const items = groupElement.querySelectorAll(':scope > .condition-items > .condition-item');
    items.forEach(item => {
        const factSelect = item.querySelector('.fact-select');
        const operatorSelect = item.querySelector('.operator-select');
        const valueInput = item.querySelector('.value-input');
        
        if (factSelect.value && operatorSelect.value) {
            // Convert value to appropriate type
            let value = valueInput.value;
            if (value === 'true') {
                value = true;
            } else if (value === 'false') {
                value = false;
            } else if (!isNaN(value) && value !== '') {
                value = Number(value);
            }
            
            result[type].push({
                fact: factSelect.value,
                operator: operatorSelect.value,
                value: value
            });
        }
    });
    
    // Process nested condition groups
    const nestedGroups = groupElement.querySelectorAll(':scope > .condition-items > .condition-group');
    nestedGroups.forEach(nestedGroup => {
        result[type].push(buildConditionGroupObject(nestedGroup));
    });
    
    return result;
}

// Update visual builder from conditions object
function updateVisualBuilderFromConditions(conditions) {
    // Clear current builder
    const builderContainer = document.getElementById('condition-builder');
    builderContainer.innerHTML = '';
    
    // Create root group
    const rootGroupType = Object.keys(conditions)[0]; // 'all' or 'any'
    const html = `
        <div class="condition-group">
            <div class="condition-group-header">
                <select class="form-select condition-group-type">
                    <option value="all" ${rootGroupType === 'all' ? 'selected' : ''}>ALL of these conditions (AND)</option>
                    <option value="any" ${rootGroupType === 'any' ? 'selected' : ''}>ANY of these conditions (OR)</option>
                </select>
                <button type="button" class="btn btn-sm btn-primary add-condition">
                    <i class="fas fa-plus"></i> Add Condition
                </button>
            </div>
            <div class="condition-items">
                <!-- Conditions will be added here -->
            </div>
        </div>
    `;
    
    builderContainer.innerHTML = html;
    
    // Add conditions to the root group
    const rootConditionsContainer = builderContainer.querySelector('.condition-items');
    
    if (conditions[rootGroupType] && Array.isArray(conditions[rootGroupType])) {
        conditions[rootGroupType].forEach(condition => {
            addConditionToVisualBuilder(rootConditionsContainer, condition);
        });
    }
}

// Add a condition to the visual builder
function addConditionToVisualBuilder(container, condition) {
    // Check if it's a group condition (has 'all' or 'any' property)
    if (condition.all || condition.any) {
        const groupType = condition.all ? 'all' : 'any';
        
        // Create a nested group
        const html = `
            <div class="condition-group mt-3">
                <div class="condition-group-header">
                    <select class="form-select condition-group-type">
                        <option value="all" ${groupType === 'all' ? 'selected' : ''}>ALL of these conditions (AND)</option>
                        <option value="any" ${groupType === 'any' ? 'selected' : ''}>ANY of these conditions (OR)</option>
                    </select>
                    <button type="button" class="btn btn-sm btn-primary add-condition">
                        <i class="fas fa-plus"></i> Add Condition
                    </button>
                </div>
                <div class="condition-items">
                    <!-- Conditions will be added here -->
                </div>
            </div>
        `;
        
        // Create temporary element to hold the HTML
        const temp = document.createElement('div');
        temp.innerHTML = html;
        
        // Add to container
        container.appendChild(temp.firstElementChild);
        
        // Process nested conditions
        const nestedContainer = container.lastElementChild.querySelector('.condition-items');
        condition[groupType].forEach(nestedCondition => {
            addConditionToVisualBuilder(nestedContainer, nestedCondition);
        });
    } else {
        // Simple condition
        addConditionItem(container);
        
        // Set values
        const item = container.lastElementChild;
        if (item) {
            const factSelect = item.querySelector('.fact-select');
            const operatorSelect = item.querySelector('.operator-select');
            const valueInput = item.querySelector('.value-input');
            
            if (factSelect && operatorSelect && valueInput) {
                if (condition.fact && factSelect.querySelector(`option[value="${condition.fact}"]`)) {
                    factSelect.value = condition.fact;
                }
                
                if (condition.operator && operatorSelect.querySelector(`option[value="${condition.operator}"]`)) {
                    operatorSelect.value = condition.operator;
                }
                
                if (condition.value !== undefined) {
                    valueInput.value = condition.value;
                }
            }
        }
    }
}

// Show toast message
function showToast(message, type = 'info') {
    let backgroundColor = '#4361ee'; // info (primary)
    
    if (type === 'success') {
        backgroundColor = '#2ecc71';
    } else if (type === 'error') {
        backgroundColor = '#e74c3c';
    } else if (type === 'warning') {
        backgroundColor = '#f39c12';
    }
    
    Toastify({
        text: message,
        duration: 3000,
        close: true,
        gravity: 'top',
        position: 'right',
        backgroundColor,
        stopOnFocus: true
    }).showToast();
}
