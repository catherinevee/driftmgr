// Form Management JavaScript for DriftMgr Dashboard

class FormManager {
    constructor() {
        this.currentForm = null;
        this.formData = {};
        this.tags = new Map();
        this.init();
    }

    init() {
        this.setupFormEventListeners();
        this.setupTagsInput();
        this.loadFormData();
    }

    setupFormEventListeners() {
        // Backend form
        document.getElementById('backend-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleBackendFormSubmit();
        });

        // State form
        document.getElementById('state-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleStateFormSubmit();
        });

        // Resource import form
        document.getElementById('resource-import-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleResourceImportSubmit();
        });

        // Resource move form
        document.getElementById('resource-move-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleResourceMoveSubmit();
        });

        // State lock form
        document.getElementById('state-lock-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleStateLockSubmit();
        });

        // Remediation form
        document.getElementById('remediation-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleRemediationFormSubmit();
        });

        // Remediation job form
        document.getElementById('remediation-job-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleRemediationJobSubmit();
        });

        // Schedule toggle
        document.getElementById('job-schedule')?.addEventListener('change', (e) => {
            this.toggleScheduleConfig(e.target.value);
        });
    }

    setupTagsInput() {
        // Setup tags input for all forms
        const tagInputs = document.querySelectorAll('.tags-input input');
        tagInputs.forEach(input => {
            input.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    this.addTag(input);
                }
            });
        });
    }

    loadFormData() {
        // Load backends for state form
        this.loadBackends();
        // Load state files for various forms
        this.loadStateFiles();
        // Load remediation strategies
        this.loadRemediationStrategies();
        // Load resources for remediation jobs
        this.loadResources();
    }

    // Backend Form Management
    openBackendForm(backendId = null) {
        this.currentForm = 'backend';
        const modal = document.getElementById('backend-form-modal');
        const title = document.getElementById('backend-form-title');
        
        if (backendId) {
            title.textContent = 'Edit Backend';
            this.loadBackendData(backendId);
        } else {
            title.textContent = 'Add New Backend';
            this.clearBackendForm();
        }
        
        modal.classList.add('show');
    }

    closeBackendForm() {
        const modal = document.getElementById('backend-form-modal');
        modal.classList.remove('show');
        this.clearBackendForm();
    }

    clearBackendForm() {
        document.getElementById('backend-form').reset();
        this.clearTags('backend-tags-list');
        this.hideAllBackendConfigs();
    }

    toggleBackendConfig() {
        const type = document.getElementById('backend-type').value;
        this.hideAllBackendConfigs();
        
        if (type) {
            const configDiv = document.getElementById(`${type}-config`);
            if (configDiv) {
                configDiv.style.display = 'block';
            }
        }
    }

    hideAllBackendConfigs() {
        const configs = ['s3-config', 'azurerm-config', 'gcs-config', 'local-config'];
        configs.forEach(id => {
            const element = document.getElementById(id);
            if (element) element.style.display = 'none';
        });
    }

    async handleBackendFormSubmit() {
        const formData = this.getFormData('backend-form');
        
        try {
            this.setFormLoading('backend-form', true);
            
            if (this.isEditMode()) {
                await this.updateBackend(formData);
            } else {
                await this.createBackend(formData);
            }
            
            this.closeBackendForm();
            this.showSuccess('Backend saved successfully');
            this.refreshBackendList();
            
        } catch (error) {
            this.showError('Failed to save backend: ' + error.message);
        } finally {
            this.setFormLoading('backend-form', false);
        }
    }

    // State Form Management
    openStateForm() {
        this.currentForm = 'state';
        const modal = document.getElementById('state-form-modal');
        modal.classList.add('show');
    }

    closeStateForm() {
        const modal = document.getElementById('state-form-modal');
        modal.classList.remove('show');
        document.getElementById('state-form').reset();
        this.clearTags('state-tags-list');
    }

    async handleStateFormSubmit() {
        const formData = this.getFormData('state-form');
        
        try {
            this.setFormLoading('state-form', true);
            await this.importStateFile(formData);
            this.closeStateForm();
            this.showSuccess('State file imported successfully');
            this.refreshStateList();
            
        } catch (error) {
            this.showError('Failed to import state file: ' + error.message);
        } finally {
            this.setFormLoading('state-form', false);
        }
    }

    // Resource Import Form Management
    openResourceImportForm() {
        this.currentForm = 'resource-import';
        const modal = document.getElementById('resource-import-modal');
        modal.classList.add('show');
    }

    closeResourceImportForm() {
        const modal = document.getElementById('resource-import-modal');
        modal.classList.remove('show');
        document.getElementById('resource-import-form').reset();
    }

    async handleResourceImportSubmit() {
        const formData = this.getFormData('resource-import-form');
        
        try {
            this.setFormLoading('resource-import-form', true);
            await this.importResource(formData);
            this.closeResourceImportForm();
            this.showSuccess('Resource imported successfully');
            this.refreshResourceList();
            
        } catch (error) {
            this.showError('Failed to import resource: ' + error.message);
        } finally {
            this.setFormLoading('resource-import-form', false);
        }
    }

    // Resource Move Form Management
    openResourceMoveForm() {
        this.currentForm = 'resource-move';
        const modal = document.getElementById('resource-move-modal');
        modal.classList.add('show');
    }

    closeResourceMoveForm() {
        const modal = document.getElementById('resource-move-modal');
        modal.classList.remove('show');
        document.getElementById('resource-move-form').reset();
    }

    async handleResourceMoveSubmit() {
        const formData = this.getFormData('resource-move-form');
        
        try {
            this.setFormLoading('resource-move-form', true);
            await this.moveResource(formData);
            this.closeResourceMoveForm();
            this.showSuccess('Resource moved successfully');
            this.refreshResourceList();
            
        } catch (error) {
            this.showError('Failed to move resource: ' + error.message);
        } finally {
            this.setFormLoading('resource-move-form', false);
        }
    }

    // State Lock Form Management
    openStateLockForm() {
        this.currentForm = 'state-lock';
        const modal = document.getElementById('state-lock-modal');
        modal.classList.add('show');
    }

    closeStateLockForm() {
        const modal = document.getElementById('state-lock-modal');
        modal.classList.remove('show');
        document.getElementById('state-lock-form').reset();
    }

    async handleStateLockSubmit() {
        const formData = this.getFormData('state-lock-form');
        
        try {
            this.setFormLoading('state-lock-form', true);
            await this.lockStateFile(formData);
            this.closeStateLockForm();
            this.showSuccess('State file locked successfully');
            this.refreshStateList();
            
        } catch (error) {
            this.showError('Failed to lock state file: ' + error.message);
        } finally {
            this.setFormLoading('state-lock-form', false);
        }
    }

    // Remediation Form Management
    openRemediationForm(strategyId = null) {
        this.currentForm = 'remediation';
        const modal = document.getElementById('remediation-form-modal');
        const title = document.getElementById('remediation-form-title');
        
        if (strategyId) {
            title.textContent = 'Edit Remediation Strategy';
            this.loadRemediationData(strategyId);
        } else {
            title.textContent = 'Create Remediation Strategy';
            this.clearRemediationForm();
        }
        
        modal.classList.add('show');
    }

    closeRemediationForm() {
        const modal = document.getElementById('remediation-form-modal');
        modal.classList.remove('show');
        this.clearRemediationForm();
    }

    clearRemediationForm() {
        document.getElementById('remediation-form').reset();
        this.clearTags('remediation-tags-list');
        this.hideAllRemediationConfigs();
    }

    toggleRemediationConfig() {
        const type = document.getElementById('remediation-type').value;
        this.hideAllRemediationConfigs();
        
        if (type) {
            const configDiv = document.getElementById(`${type.replace('_', '-')}-config`);
            if (configDiv) {
                configDiv.style.display = 'block';
            }
        }
    }

    hideAllRemediationConfigs() {
        const configs = ['terraform-apply-config', 'terraform-import-config', 
                        'terraform-remove-config', 'terraform-move-config', 
                        'manual-config', 'script-config'];
        configs.forEach(id => {
            const element = document.getElementById(id);
            if (element) element.style.display = 'none';
        });
    }

    async handleRemediationFormSubmit() {
        const formData = this.getFormData('remediation-form');
        
        try {
            this.setFormLoading('remediation-form', true);
            
            if (this.isEditMode()) {
                await this.updateRemediationStrategy(formData);
            } else {
                await this.createRemediationStrategy(formData);
            }
            
            this.closeRemediationForm();
            this.showSuccess('Remediation strategy saved successfully');
            this.refreshRemediationList();
            
        } catch (error) {
            this.showError('Failed to save remediation strategy: ' + error.message);
        } finally {
            this.setFormLoading('remediation-form', false);
        }
    }

    // Remediation Job Form Management
    openRemediationJobForm() {
        this.currentForm = 'remediation-job';
        const modal = document.getElementById('remediation-job-modal');
        modal.classList.add('show');
    }

    closeRemediationJobForm() {
        const modal = document.getElementById('remediation-job-modal');
        modal.classList.remove('show');
        document.getElementById('remediation-job-form').reset();
        this.clearResourceSelection();
    }

    toggleScheduleConfig(schedule) {
        const configDiv = document.getElementById('schedule-config');
        if (schedule === 'scheduled') {
            configDiv.style.display = 'block';
        } else {
            configDiv.style.display = 'none';
        }
    }

    async handleRemediationJobSubmit() {
        const formData = this.getFormData('remediation-job-form');
        const selectedResources = this.getSelectedResources();
        
        if (selectedResources.length === 0) {
            this.showError('Please select at least one resource');
            return;
        }
        
        formData.resources = selectedResources;
        
        try {
            this.setFormLoading('remediation-job-form', true);
            await this.createRemediationJob(formData);
            this.closeRemediationJobForm();
            this.showSuccess('Remediation job created successfully');
            this.refreshRemediationJobList();
            
        } catch (error) {
            this.showError('Failed to create remediation job: ' + error.message);
        } finally {
            this.setFormLoading('remediation-job-form', false);
        }
    }

    // Tags Management
    addTag(input) {
        const tag = input.value.trim();
        if (tag && !this.tags.has(tag)) {
            this.tags.set(tag, tag);
            this.renderTags(input.id.replace('-tags', '-tags-list'));
            input.value = '';
        }
    }

    removeTag(tag, listId) {
        this.tags.delete(tag);
        this.renderTags(listId);
    }

    renderTags(listId) {
        const list = document.getElementById(listId);
        if (!list) return;
        
        list.innerHTML = '';
        this.tags.forEach(tag => {
            const tagElement = document.createElement('div');
            tagElement.className = 'tag';
            tagElement.innerHTML = `
                ${tag}
                <button type="button" class="tag-remove" onclick="formManager.removeTag('${tag}', '${listId}')">
                    <i class="fas fa-times"></i>
                </button>
            `;
            list.appendChild(tagElement);
        });
    }

    clearTags(listId) {
        this.tags.clear();
        this.renderTags(listId);
    }

    // Resource Selection
    selectAllResources() {
        const checkboxes = document.querySelectorAll('#job-resource-list input[type="checkbox"]');
        checkboxes.forEach(checkbox => {
            checkbox.checked = true;
        });
    }

    clearAllResources() {
        const checkboxes = document.querySelectorAll('#job-resource-list input[type="checkbox"]');
        checkboxes.forEach(checkbox => {
            checkbox.checked = false;
        });
    }

    getSelectedResources() {
        const checkboxes = document.querySelectorAll('#job-resource-list input[type="checkbox"]:checked');
        return Array.from(checkboxes).map(checkbox => checkbox.value);
    }

    // Backend Test Connection
    openBackendTest(backendId) {
        const modal = document.getElementById('backend-test-modal');
        const backendName = document.getElementById('test-backend-name');
        const backendDetails = document.getElementById('test-backend-details');
        
        // Load backend details
        this.loadBackendDetails(backendId).then(backend => {
            backendName.textContent = backend.name;
            backendDetails.textContent = `${backend.type} - ${backend.config?.bucket || backend.config?.container_name || 'N/A'}`;
        });
        
        modal.classList.add('show');
    }

    closeBackendTest() {
        const modal = document.getElementById('backend-test-modal');
        modal.classList.remove('show');
        document.getElementById('test-results').style.display = 'none';
    }

    async testBackendConnection() {
        const resultsDiv = document.getElementById('test-results');
        const statusDiv = document.getElementById('test-status');
        const detailsDiv = document.getElementById('test-details');
        
        resultsDiv.style.display = 'block';
        statusDiv.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Testing...';
        statusDiv.className = 'test-status loading';
        
        try {
            // Simulate API call
            await new Promise(resolve => setTimeout(resolve, 2000));
            
            statusDiv.innerHTML = '<i class="fas fa-check"></i> Connection Successful';
            statusDiv.className = 'test-status success';
            detailsDiv.textContent = 'Backend is accessible and credentials are valid.';
            
        } catch (error) {
            statusDiv.innerHTML = '<i class="fas fa-times"></i> Connection Failed';
            statusDiv.className = 'test-status error';
            detailsDiv.textContent = error.message;
        }
    }

    // Utility Methods
    getFormData(formId) {
        const form = document.getElementById(formId);
        const formData = new FormData(form);
        const data = {};
        
        for (let [key, value] of formData.entries()) {
            if (key.includes('.')) {
                const [parent, child] = key.split('.');
                if (!data[parent]) data[parent] = {};
                data[parent][child] = value;
            } else {
                data[key] = value;
            }
        }
        
        // Add tags
        data.tags = Array.from(this.tags.keys());
        
        return data;
    }

    setFormLoading(formId, loading) {
        const form = document.getElementById(formId);
        const submitBtn = form.querySelector('button[type="submit"]');
        
        if (loading) {
            form.classList.add('loading');
            submitBtn.disabled = true;
            submitBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Saving...';
        } else {
            form.classList.remove('loading');
            submitBtn.disabled = false;
            submitBtn.innerHTML = submitBtn.innerHTML.replace('Saving...', 'Save');
        }
    }

    isEditMode() {
        return this.currentForm && document.getElementById(`${this.currentForm}-form-title`).textContent.includes('Edit');
    }

    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showError(message) {
        this.showNotification(message, 'error');
    }

    showNotification(message, type) {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <i class="fas fa-${type === 'success' ? 'check' : 'times'}"></i>
            <span>${message}</span>
        `;
        
        // Add to page
        document.body.appendChild(notification);
        
        // Remove after 5 seconds
        setTimeout(() => {
            notification.remove();
        }, 5000);
    }

    // API Methods (Mock implementations)
    async loadBackends() {
        // Mock implementation - in real app, this would call the API
        const backends = [
            { id: 'backend-1', name: 'Production S3 Backend', type: 's3' },
            { id: 'backend-2', name: 'Development Azure Backend', type: 'azurerm' }
        ];
        
        this.populateSelect('state-backend', backends);
        this.populateSelect('import-state-path', backends);
        this.populateSelect('move-state-path', backends);
        this.populateSelect('lock-state-path', backends);
    }

    async loadStateFiles() {
        // Mock implementation
        const stateFiles = [
            { id: 'state-1', name: 'terraform.tfstate', path: 'terraform.tfstate' },
            { id: 'state-2', name: 'prod/terraform.tfstate', path: 'prod/terraform.tfstate' }
        ];
        
        this.populateSelect('import-state-path', stateFiles);
        this.populateSelect('move-state-path', stateFiles);
        this.populateSelect('lock-state-path', stateFiles);
    }

    async loadRemediationStrategies() {
        // Mock implementation
        const strategies = [
            { id: 'strategy-1', name: 'Auto-fix S3 Buckets', type: 'terraform_apply' },
            { id: 'strategy-2', name: 'Import Missing Resources', type: 'terraform_import' }
        ];
        
        this.populateSelect('job-strategy', strategies);
    }

    async loadResources() {
        // Mock implementation
        const resources = [
            { id: 'resource-1', name: 'web-server-01', type: 'aws_instance' },
            { id: 'resource-2', name: 'data-bucket', type: 'aws_s3_bucket' }
        ];
        
        this.renderResourceList(resources);
    }

    populateSelect(selectId, options) {
        const select = document.getElementById(selectId);
        if (!select) return;
        
        // Clear existing options (except first)
        while (select.children.length > 1) {
            select.removeChild(select.lastChild);
        }
        
        // Add new options
        options.forEach(option => {
            const optionElement = document.createElement('option');
            optionElement.value = option.id;
            optionElement.textContent = option.name;
            select.appendChild(optionElement);
        });
    }

    renderResourceList(resources) {
        const list = document.getElementById('job-resource-list');
        if (!list) return;
        
        list.innerHTML = '';
        resources.forEach(resource => {
            const item = document.createElement('div');
            item.className = 'resource-item';
            item.innerHTML = `
                <input type="checkbox" value="${resource.id}">
                <div class="resource-info">
                    <div class="resource-name">${resource.name}</div>
                    <div class="resource-type">${resource.type}</div>
                </div>
            `;
            list.appendChild(item);
        });
    }

    // Mock API calls
    async createBackend(data) {
        console.log('Creating backend:', data);
        await new Promise(resolve => setTimeout(resolve, 1000));
    }

    async updateBackend(data) {
        console.log('Updating backend:', data);
        await new Promise(resolve => setTimeout(resolve, 1000));
    }

    async importStateFile(data) {
        console.log('Importing state file:', data);
        await new Promise(resolve => setTimeout(resolve, 1000));
    }

    async importResource(data) {
        console.log('Importing resource:', data);
        await new Promise(resolve => setTimeout(resolve, 1000));
    }

    async moveResource(data) {
        console.log('Moving resource:', data);
        await new Promise(resolve => setTimeout(resolve, 1000));
    }

    async lockStateFile(data) {
        console.log('Locking state file:', data);
        await new Promise(resolve => setTimeout(resolve, 1000));
    }

    async createRemediationStrategy(data) {
        console.log('Creating remediation strategy:', data);
        await new Promise(resolve => setTimeout(resolve, 1000));
    }

    async updateRemediationStrategy(data) {
        console.log('Updating remediation strategy:', data);
        await new Promise(resolve => setTimeout(resolve, 1000));
    }

    async createRemediationJob(data) {
        console.log('Creating remediation job:', data);
        await new Promise(resolve => setTimeout(resolve, 1000));
    }

    // Refresh methods (mock implementations)
    refreshBackendList() {
        console.log('Refreshing backend list');
    }

    refreshStateList() {
        console.log('Refreshing state list');
    }

    refreshResourceList() {
        console.log('Refreshing resource list');
    }

    refreshRemediationList() {
        console.log('Refreshing remediation list');
    }

    refreshRemediationJobList() {
        console.log('Refreshing remediation job list');
    }
}

// Global form manager instance
const formManager = new FormManager();

// Global functions for HTML onclick handlers
function openBackendForm(backendId = null) {
    formManager.openBackendForm(backendId);
}

function closeBackendForm() {
    formManager.closeBackendForm();
}

function toggleBackendConfig() {
    formManager.toggleBackendConfig();
}

function openStateForm() {
    formManager.openStateForm();
}

function closeStateForm() {
    formManager.closeStateForm();
}

function openResourceImportForm() {
    formManager.openResourceImportForm();
}

function closeResourceImportForm() {
    formManager.closeResourceImportForm();
}

function openResourceMoveForm() {
    formManager.openResourceMoveForm();
}

function closeResourceMoveForm() {
    formManager.closeResourceMoveForm();
}

function openStateLockForm() {
    formManager.openStateLockForm();
}

function closeStateLockForm() {
    formManager.closeStateLockForm();
}

function openRemediationForm(strategyId = null) {
    formManager.openRemediationForm(strategyId);
}

function closeRemediationForm() {
    formManager.closeRemediationForm();
}

function toggleRemediationConfig() {
    formManager.toggleRemediationConfig();
}

function openRemediationJobForm() {
    formManager.openRemediationJobForm();
}

function closeRemediationJobForm() {
    formManager.closeRemediationJobForm();
}

function openBackendTest(backendId) {
    formManager.openBackendTest(backendId);
}

function closeBackendTest() {
    formManager.closeBackendTest();
}

function testBackendConnection() {
    formManager.testBackendConnection();
}

function selectAllResources() {
    formManager.selectAllResources();
}

function clearAllResources() {
    formManager.clearAllResources();
}
