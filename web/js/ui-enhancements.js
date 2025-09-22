// UI Enhancement Components for DriftMgr
// This file contains enhanced UI components to replace TODO implementations

class BackendConfigurationUI {
    constructor(app) {
        this.app = app;
        this.modal = null;
        this.currentConfig = null;
    }

    // Initialize backend configuration modal
    init() {
        this.createModal();
        this.bindEvents();
    }

    // Create the backend configuration modal
    createModal() {
        const modalHTML = `
            <div id="backendConfigModal" class="modal fade" tabindex="-1" role="dialog">
                <div class="modal-dialog modal-lg" role="document">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">Backend Configuration</h5>
                            <button type="button" class="close" data-dismiss="modal">
                                <span>&times;</span>
                            </button>
                        </div>
                        <div class="modal-body">
                            <form id="backendConfigForm">
                                <div class="form-group">
                                    <label for="backendType">Backend Type</label>
                                    <select class="form-control" id="backendType" required>
                                        <option value="">Select Backend Type</option>
                                        <option value="s3">Amazon S3</option>
                                        <option value="gcs">Google Cloud Storage</option>
                                        <option value="azurerm">Azure Storage</option>
                                        <option value="remote">Terraform Cloud</option>
                                        <option value="local">Local</option>
                                    </select>
                                </div>
                                
                                <div id="backendConfigFields">
                                    <!-- Dynamic fields based on backend type -->
                                </div>
                                
                                <div class="form-group">
                                    <div class="form-check">
                                        <input class="form-check-input" type="checkbox" id="enableEncryption">
                                        <label class="form-check-label" for="enableEncryption">
                                            Enable Encryption
                                        </label>
                                    </div>
                                </div>
                                
                                <div class="form-group">
                                    <div class="form-check">
                                        <input class="form-check-input" type="checkbox" id="enableVersioning">
                                        <label class="form-check-label" for="enableVersioning">
                                            Enable State Versioning
                                        </label>
                                    </div>
                                </div>
                            </form>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-dismiss="modal">Cancel</button>
                            <button type="button" class="btn btn-primary" id="testConnectionBtn">Test Connection</button>
                            <button type="button" class="btn btn-success" id="saveConfigBtn">Save Configuration</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modalHTML);
        this.modal = document.getElementById('backendConfigModal');
    }

    // Bind event handlers
    bindEvents() {
        const backendType = document.getElementById('backendType');
        const testBtn = document.getElementById('testConnectionBtn');
        const saveBtn = document.getElementById('saveConfigBtn');

        backendType.addEventListener('change', (e) => {
            this.updateConfigFields(e.target.value);
        });

        testBtn.addEventListener('click', () => {
            this.testConnection();
        });

        saveBtn.addEventListener('click', () => {
            this.saveConfiguration();
        });
    }

    // Update configuration fields based on backend type
    updateConfigFields(backendType) {
        const fieldsContainer = document.getElementById('backendConfigFields');
        let fieldsHTML = '';

        switch (backendType) {
            case 's3':
                fieldsHTML = `
                    <div class="form-group">
                        <label for="s3Bucket">S3 Bucket</label>
                        <input type="text" class="form-control" id="s3Bucket" required>
                    </div>
                    <div class="form-group">
                        <label for="s3Key">State Key</label>
                        <input type="text" class="form-control" id="s3Key" value="terraform.tfstate">
                    </div>
                    <div class="form-group">
                        <label for="s3Region">AWS Region</label>
                        <input type="text" class="form-control" id="s3Region" required>
                    </div>
                    <div class="form-group">
                        <label for="dynamodbTable">DynamoDB Table (for locking)</label>
                        <input type="text" class="form-control" id="dynamodbTable">
                    </div>
                `;
                break;
            case 'gcs':
                fieldsHTML = `
                    <div class="form-group">
                        <label for="gcsBucket">GCS Bucket</label>
                        <input type="text" class="form-control" id="gcsBucket" required>
                    </div>
                    <div class="form-group">
                        <label for="gcsPrefix">Prefix</label>
                        <input type="text" class="form-control" id="gcsPrefix" value="terraform/state">
                    </div>
                    <div class="form-group">
                        <label for="gcsProject">Project ID</label>
                        <input type="text" class="form-control" id="gcsProject" required>
                    </div>
                `;
                break;
            case 'azurerm':
                fieldsHTML = `
                    <div class="form-group">
                        <label for="azureStorageAccount">Storage Account</label>
                        <input type="text" class="form-control" id="azureStorageAccount" required>
                    </div>
                    <div class="form-group">
                        <label for="azureContainer">Container Name</label>
                        <input type="text" class="form-control" id="azureContainer" required>
                    </div>
                    <div class="form-group">
                        <label for="azureKey">State Key</label>
                        <input type="text" class="form-control" id="azureKey" value="terraform.tfstate">
                    </div>
                    <div class="form-group">
                        <label for="azureResourceGroup">Resource Group</label>
                        <input type="text" class="form-control" id="azureResourceGroup" required>
                    </div>
                `;
                break;
            case 'remote':
                fieldsHTML = `
                    <div class="form-group">
                        <label for="tfcHostname">Terraform Cloud Hostname</label>
                        <input type="text" class="form-control" id="tfcHostname" value="app.terraform.io">
                    </div>
                    <div class="form-group">
                        <label for="tfcOrganization">Organization</label>
                        <input type="text" class="form-control" id="tfcOrganization" required>
                    </div>
                    <div class="form-group">
                        <label for="tfcWorkspace">Workspace</label>
                        <input type="text" class="form-control" id="tfcWorkspace" required>
                    </div>
                    <div class="form-group">
                        <label for="tfcToken">API Token</label>
                        <input type="password" class="form-control" id="tfcToken" required>
                    </div>
                `;
                break;
            case 'local':
                fieldsHTML = `
                    <div class="form-group">
                        <label for="localPath">Local Path</label>
                        <input type="text" class="form-control" id="localPath" value="./terraform.tfstate">
                    </div>
                `;
                break;
        }

        fieldsContainer.innerHTML = fieldsHTML;
    }

    // Test backend connection
    async testConnection() {
        const testBtn = document.getElementById('testConnectionBtn');
        const originalText = testBtn.textContent;
        
        testBtn.textContent = 'Testing...';
        testBtn.disabled = true;

        try {
            const config = this.getFormData();
            const response = await fetch('/api/v1/backends/test', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(config)
            });

            if (response.ok) {
                this.showAlert('Connection test successful!', 'success');
            } else {
                const error = await response.json();
                this.showAlert(`Connection test failed: ${error.message}`, 'danger');
            }
        } catch (error) {
            this.showAlert(`Connection test failed: ${error.message}`, 'danger');
        } finally {
            testBtn.textContent = originalText;
            testBtn.disabled = false;
        }
    }

    // Save backend configuration
    async saveConfiguration() {
        try {
            const config = this.getFormData();
            const response = await fetch('/api/v1/backends/config', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(config)
            });

            if (response.ok) {
                this.showAlert('Configuration saved successfully!', 'success');
                $(this.modal).modal('hide');
                // Refresh the app state
                if (this.app && this.app.loadState) {
                    this.app.loadState();
                }
            } else {
                const error = await response.json();
                this.showAlert(`Failed to save configuration: ${error.message}`, 'danger');
            }
        } catch (error) {
            this.showAlert(`Failed to save configuration: ${error.message}`, 'danger');
        }
    }

    // Get form data
    getFormData() {
        const backendType = document.getElementById('backendType').value;
        const config = {
            type: backendType,
            config: {}
        };

        // Get type-specific configuration
        switch (backendType) {
            case 's3':
                config.config = {
                    bucket: document.getElementById('s3Bucket').value,
                    key: document.getElementById('s3Key').value,
                    region: document.getElementById('s3Region').value,
                    dynamodb_table: document.getElementById('dynamodbTable').value
                };
                break;
            case 'gcs':
                config.config = {
                    bucket: document.getElementById('gcsBucket').value,
                    prefix: document.getElementById('gcsPrefix').value,
                    project_id: document.getElementById('gcsProject').value
                };
                break;
            case 'azurerm':
                config.config = {
                    storage_account_name: document.getElementById('azureStorageAccount').value,
                    container_name: document.getElementById('azureContainer').value,
                    key: document.getElementById('azureKey').value,
                    resource_group_name: document.getElementById('azureResourceGroup').value
                };
                break;
            case 'remote':
                config.config = {
                    hostname: document.getElementById('tfcHostname').value,
                    organization: document.getElementById('tfcOrganization').value,
                    workspace: document.getElementById('tfcWorkspace').value,
                    token: document.getElementById('tfcToken').value
                };
                break;
            case 'local':
                config.config = {
                    path: document.getElementById('localPath').value
                };
                break;
        }

        // Add common options
        config.config.encrypt = document.getElementById('enableEncryption').checked;
        config.config.versioning = document.getElementById('enableVersioning').checked;

        return config;
    }

    // Show alert message
    showAlert(message, type) {
        const alertHTML = `
            <div class="alert alert-${type} alert-dismissible fade show" role="alert">
                ${message}
                <button type="button" class="close" data-dismiss="alert">
                    <span>&times;</span>
                </button>
            </div>
        `;
        
        const modalBody = this.modal.querySelector('.modal-body');
        modalBody.insertAdjacentHTML('afterbegin', alertHTML);
        
        // Auto-dismiss after 5 seconds
        setTimeout(() => {
            const alert = modalBody.querySelector('.alert');
            if (alert) {
                alert.remove();
            }
        }, 5000);
    }

    // Open the modal
    open() {
        $(this.modal).modal('show');
    }
}

class ResourceMoveUI {
    constructor(app) {
        this.app = app;
        this.modal = null;
    }

    // Initialize resource move modal
    init() {
        this.createModal();
        this.bindEvents();
    }

    // Create the resource move modal
    createModal() {
        const modalHTML = `
            <div id="resourceMoveModal" class="modal fade" tabindex="-1" role="dialog">
                <div class="modal-dialog modal-lg" role="document">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">Move Resource</h5>
                            <button type="button" class="close" data-dismiss="modal">
                                <span>&times;</span>
                            </button>
                        </div>
                        <div class="modal-body">
                            <form id="resourceMoveForm">
                                <div class="form-group">
                                    <label for="sourceResource">Source Resource</label>
                                    <select class="form-control" id="sourceResource" required>
                                        <option value="">Select a resource to move</option>
                                    </select>
                                </div>
                                
                                <div class="form-group">
                                    <label for="targetModule">Target Module</label>
                                    <input type="text" class="form-control" id="targetModule" placeholder="e.g., module.new_module">
                                </div>
                                
                                <div class="form-group">
                                    <label for="targetName">Target Name</label>
                                    <input type="text" class="form-control" id="targetName" placeholder="New resource name">
                                </div>
                                
                                <div class="alert alert-info">
                                    <strong>Note:</strong> This operation will update the Terraform state file to reflect the new resource location.
                                    Make sure to update your Terraform configuration files accordingly.
                                </div>
                            </form>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-dismiss="modal">Cancel</button>
                            <button type="button" class="btn btn-warning" id="previewMoveBtn">Preview Move</button>
                            <button type="button" class="btn btn-primary" id="executeMoveBtn">Execute Move</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modalHTML);
        this.modal = document.getElementById('resourceMoveModal');
    }

    // Bind event handlers
    bindEvents() {
        const previewBtn = document.getElementById('previewMoveBtn');
        const executeBtn = document.getElementById('executeMoveBtn');

        previewBtn.addEventListener('click', () => {
            this.previewMove();
        });

        executeBtn.addEventListener('click', () => {
            this.executeMove();
        });
    }

    // Load resources into the dropdown
    async loadResources() {
        const select = document.getElementById('sourceResource');
        select.innerHTML = '<option value="">Loading resources...</option>';

        try {
            const response = await fetch('/api/v1/state/resources');
            if (!response.ok) throw new Error('Failed to load resources');

            const resources = await response.json();
            select.innerHTML = '<option value="">Select a resource to move</option>';

            resources.forEach(resource => {
                const option = document.createElement('option');
                option.value = resource.address;
                option.textContent = `${resource.type}.${resource.name} (${resource.module || 'root'})`;
                select.appendChild(option);
            });
        } catch (error) {
            select.innerHTML = '<option value="">Error loading resources</option>';
            console.error('Failed to load resources:', error);
        }
    }

    // Preview the move operation
    async previewMove() {
        const sourceResource = document.getElementById('sourceResource').value;
        const targetModule = document.getElementById('targetModule').value;
        const targetName = document.getElementById('targetName').value;

        if (!sourceResource) {
            this.showAlert('Please select a source resource', 'warning');
            return;
        }

        try {
            const moveData = {
                source: sourceResource,
                target_module: targetModule,
                target_name: targetName
            };

            const response = await fetch('/api/v1/state/move/preview', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(moveData)
            });

            if (response.ok) {
                const preview = await response.json();
                this.showMovePreview(preview);
            } else {
                const error = await response.json();
                this.showAlert(`Preview failed: ${error.message}`, 'danger');
            }
        } catch (error) {
            this.showAlert(`Preview failed: ${error.message}`, 'danger');
        }
    }

    // Execute the move operation
    async executeMove() {
        const sourceResource = document.getElementById('sourceResource').value;
        const targetModule = document.getElementById('targetModule').value;
        const targetName = document.getElementById('targetName').value;

        if (!sourceResource) {
            this.showAlert('Please select a source resource', 'warning');
            return;
        }

        if (!confirm('Are you sure you want to move this resource? This action cannot be undone.')) {
            return;
        }

        try {
            const moveData = {
                source: sourceResource,
                target_module: targetModule,
                target_name: targetName
            };

            const response = await fetch('/api/v1/state/move', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(moveData)
            });

            if (response.ok) {
                this.showAlert('Resource moved successfully!', 'success');
                $(this.modal).modal('hide');
                // Refresh the app state
                if (this.app && this.app.loadState) {
                    this.app.loadState();
                }
            } else {
                const error = await response.json();
                this.showAlert(`Move failed: ${error.message}`, 'danger');
            }
        } catch (error) {
            this.showAlert(`Move failed: ${error.message}`, 'danger');
        }
    }

    // Show move preview
    showMovePreview(preview) {
        const previewHTML = `
            <div class="card mt-3">
                <div class="card-header">
                    <h6>Move Preview</h6>
                </div>
                <div class="card-body">
                    <p><strong>From:</strong> ${preview.source}</p>
                    <p><strong>To:</strong> ${preview.target}</p>
                    <p><strong>Dependencies affected:</strong> ${preview.dependencies.length}</p>
                    ${preview.dependencies.length > 0 ? `
                        <ul>
                            ${preview.dependencies.map(dep => `<li>${dep}</li>`).join('')}
                        </ul>
                    ` : ''}
                </div>
            </div>
        `;
        
        const modalBody = this.modal.querySelector('.modal-body');
        const existingPreview = modalBody.querySelector('.card');
        if (existingPreview) {
            existingPreview.remove();
        }
        modalBody.insertAdjacentHTML('beforeend', previewHTML);
    }

    // Show alert message
    showAlert(message, type) {
        const alertHTML = `
            <div class="alert alert-${type} alert-dismissible fade show" role="alert">
                ${message}
                <button type="button" class="close" data-dismiss="alert">
                    <span>&times;</span>
                </button>
            </div>
        `;
        
        const modalBody = this.modal.querySelector('.modal-body');
        modalBody.insertAdjacentHTML('afterbegin', alertHTML);
        
        // Auto-dismiss after 5 seconds
        setTimeout(() => {
            const alert = modalBody.querySelector('.alert');
            if (alert) {
                alert.remove();
            }
        }, 5000);
    }

    // Open the modal
    async open() {
        await this.loadResources();
        $(this.modal).modal('show');
    }
}

class ResourceRemovalUI {
    constructor(app) {
        this.app = app;
        this.modal = null;
    }

    // Initialize resource removal modal
    init() {
        this.createModal();
        this.bindEvents();
    }

    // Create the resource removal modal
    createModal() {
        const modalHTML = `
            <div id="resourceRemovalModal" class="modal fade" tabindex="-1" role="dialog">
                <div class="modal-dialog modal-lg" role="document">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">Remove Resource from State</h5>
                            <button type="button" class="close" data-dismiss="modal">
                                <span>&times;</span>
                            </button>
                        </div>
                        <div class="modal-body">
                            <form id="resourceRemovalForm">
                                <div class="form-group">
                                    <label for="resourceToRemove">Resource to Remove</label>
                                    <select class="form-control" id="resourceToRemove" required>
                                        <option value="">Select a resource to remove</option>
                                    </select>
                                </div>
                                
                                <div class="alert alert-warning">
                                    <strong>Warning:</strong> This operation will remove the resource from the Terraform state file.
                                    The actual resource will not be destroyed, but Terraform will no longer manage it.
                                </div>
                                
                                <div class="form-group">
                                    <div class="form-check">
                                        <input class="form-check-input" type="checkbox" id="confirmRemoval">
                                        <label class="form-check-label" for="confirmRemoval">
                                            I understand that this will remove the resource from state management
                                        </label>
                                    </div>
                                </div>
                            </form>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-dismiss="modal">Cancel</button>
                            <button type="button" class="btn btn-danger" id="executeRemovalBtn" disabled>Remove from State</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modalHTML);
        this.modal = document.getElementById('resourceRemovalModal');
    }

    // Bind event handlers
    bindEvents() {
        const confirmCheckbox = document.getElementById('confirmRemoval');
        const executeBtn = document.getElementById('executeRemovalBtn');

        confirmCheckbox.addEventListener('change', (e) => {
            executeBtn.disabled = !e.target.checked;
        });

        executeBtn.addEventListener('click', () => {
            this.executeRemoval();
        });
    }

    // Load resources into the dropdown
    async loadResources() {
        const select = document.getElementById('resourceToRemove');
        select.innerHTML = '<option value="">Loading resources...</option>';

        try {
            const response = await fetch('/api/v1/state/resources');
            if (!response.ok) throw new Error('Failed to load resources');

            const resources = await response.json();
            select.innerHTML = '<option value="">Select a resource to remove</option>';

            resources.forEach(resource => {
                const option = document.createElement('option');
                option.value = resource.address;
                option.textContent = `${resource.type}.${resource.name} (${resource.module || 'root'})`;
                select.appendChild(option);
            });
        } catch (error) {
            select.innerHTML = '<option value="">Error loading resources</option>';
            console.error('Failed to load resources:', error);
        }
    }

    // Execute the removal operation
    async executeRemoval() {
        const resourceToRemove = document.getElementById('resourceToRemove').value;

        if (!resourceToRemove) {
            this.showAlert('Please select a resource to remove', 'warning');
            return;
        }

        if (!confirm('Are you absolutely sure you want to remove this resource from state? This action cannot be undone.')) {
            return;
        }

        try {
            const response = await fetch('/api/v1/state/remove', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ resource: resourceToRemove })
            });

            if (response.ok) {
                this.showAlert('Resource removed from state successfully!', 'success');
                $(this.modal).modal('hide');
                // Refresh the app state
                if (this.app && this.app.loadState) {
                    this.app.loadState();
                }
            } else {
                const error = await response.json();
                this.showAlert(`Removal failed: ${error.message}`, 'danger');
            }
        } catch (error) {
            this.showAlert(`Removal failed: ${error.message}`, 'danger');
        }
    }

    // Show alert message
    showAlert(message, type) {
        const alertHTML = `
            <div class="alert alert-${type} alert-dismissible fade show" role="alert">
                ${message}
                <button type="button" class="close" data-dismiss="alert">
                    <span>&times;</span>
                </button>
            </div>
        `;
        
        const modalBody = this.modal.querySelector('.modal-body');
        modalBody.insertAdjacentHTML('afterbegin', alertHTML);
        
        // Auto-dismiss after 5 seconds
        setTimeout(() => {
            const alert = modalBody.querySelector('.alert');
            if (alert) {
                alert.remove();
            }
        }, 5000);
    }

    // Open the modal
    async open() {
        await this.loadResources();
        $(this.modal).modal('show');
    }
}

class ImportWizardUI {
    constructor(app) {
        this.app = app;
        this.modal = null;
        this.currentStep = 1;
        this.totalSteps = 3;
    }

    // Initialize import wizard modal
    init() {
        this.createModal();
        this.bindEvents();
    }

    // Create the import wizard modal
    createModal() {
        const modalHTML = `
            <div id="importWizardModal" class="modal fade" tabindex="-1" role="dialog">
                <div class="modal-dialog modal-xl" role="document">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">Import Resource Wizard</h5>
                            <button type="button" class="close" data-dismiss="modal">
                                <span>&times;</span>
                            </button>
                        </div>
                        <div class="modal-body">
                            <div class="wizard-progress mb-4">
                                <div class="progress">
                                    <div class="progress-bar" role="progressbar" style="width: 33%"></div>
                                </div>
                                <div class="d-flex justify-content-between mt-2">
                                    <span class="step active">1. Resource Type</span>
                                    <span class="step">2. Resource Details</span>
                                    <span class="step">3. Import</span>
                                </div>
                            </div>
                            
                            <div id="wizardContent">
                                <!-- Dynamic content based on step -->
                            </div>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" id="prevStepBtn" disabled>Previous</button>
                            <button type="button" class="btn btn-primary" id="nextStepBtn">Next</button>
                            <button type="button" class="btn btn-success" id="importBtn" style="display: none;">Import Resource</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modalHTML);
        this.modal = document.getElementById('importWizardModal');
    }

    // Bind event handlers
    bindEvents() {
        const prevBtn = document.getElementById('prevStepBtn');
        const nextBtn = document.getElementById('nextStepBtn');
        const importBtn = document.getElementById('importBtn');

        prevBtn.addEventListener('click', () => {
            this.previousStep();
        });

        nextBtn.addEventListener('click', () => {
            this.nextStep();
        });

        importBtn.addEventListener('click', () => {
            this.executeImport();
        });
    }

    // Show step content
    showStep(step) {
        const content = document.getElementById('wizardContent');
        const progressBar = document.querySelector('.progress-bar');
        const steps = document.querySelectorAll('.step');
        
        // Update progress bar
        progressBar.style.width = `${(step / this.totalSteps) * 100}%`;
        
        // Update step indicators
        steps.forEach((stepEl, index) => {
            stepEl.classList.toggle('active', index + 1 === step);
        });

        switch (step) {
            case 1:
                this.showStep1(content);
                break;
            case 2:
                this.showStep2(content);
                break;
            case 3:
                this.showStep3(content);
                break;
        }

        this.updateButtons();
    }

    // Show step 1: Resource type selection
    showStep1(content) {
        content.innerHTML = `
            <h6>Select Resource Type</h6>
            <div class="form-group">
                <label for="resourceType">Resource Type</label>
                <select class="form-control" id="resourceType" required>
                    <option value="">Select a resource type</option>
                    <option value="aws_instance">AWS Instance</option>
                    <option value="aws_s3_bucket">AWS S3 Bucket</option>
                    <option value="aws_security_group">AWS Security Group</option>
                    <option value="google_compute_instance">GCP Compute Instance</option>
                    <option value="google_storage_bucket">GCP Storage Bucket</option>
                    <option value="azurerm_virtual_machine">Azure Virtual Machine</option>
                    <option value="azurerm_storage_account">Azure Storage Account</option>
                </select>
            </div>
            <div class="form-group">
                <label for="resourceName">Resource Name</label>
                <input type="text" class="form-control" id="resourceName" placeholder="e.g., my_instance" required>
            </div>
        `;
    }

    // Show step 2: Resource details
    showStep2(content) {
        const resourceType = document.getElementById('resourceType').value;
        const resourceName = document.getElementById('resourceName').value;

        let detailsHTML = `
            <h6>Resource Details</h6>
            <p><strong>Type:</strong> ${resourceType}</p>
            <p><strong>Name:</strong> ${resourceName}</p>
            <div class="form-group">
                <label for="resourceId">Resource ID</label>
                <input type="text" class="form-control" id="resourceId" placeholder="Enter the actual resource ID" required>
                <small class="form-text text-muted">
                    This is the unique identifier of the existing resource in your cloud provider.
                </small>
            </div>
        `;

        // Add provider-specific fields
        if (resourceType.startsWith('aws_')) {
            detailsHTML += `
                <div class="form-group">
                    <label for="awsRegion">AWS Region</label>
                    <input type="text" class="form-control" id="awsRegion" placeholder="e.g., us-west-2">
                </div>
            `;
        } else if (resourceType.startsWith('google_')) {
            detailsHTML += `
                <div class="form-group">
                    <label for="gcpProject">GCP Project</label>
                    <input type="text" class="form-control" id="gcpProject" placeholder="e.g., my-project">
                </div>
                <div class="form-group">
                    <label for="gcpZone">GCP Zone</label>
                    <input type="text" class="form-control" id="gcpZone" placeholder="e.g., us-central1-a">
                </div>
            `;
        } else if (resourceType.startsWith('azurerm_')) {
            detailsHTML += `
                <div class="form-group">
                    <label for="azureResourceGroup">Azure Resource Group</label>
                    <input type="text" class="form-control" id="azureResourceGroup" placeholder="e.g., my-resource-group">
                </div>
            `;
        }

        content.innerHTML = detailsHTML;
    }

    // Show step 3: Import confirmation
    showStep3(content) {
        const resourceType = document.getElementById('resourceType').value;
        const resourceName = document.getElementById('resourceName').value;
        const resourceId = document.getElementById('resourceId').value;

        content.innerHTML = `
            <h6>Import Confirmation</h6>
            <div class="card">
                <div class="card-body">
                    <h6>Resource to Import:</h6>
                    <p><strong>Type:</strong> ${resourceType}</p>
                    <p><strong>Name:</strong> ${resourceName}</p>
                    <p><strong>ID:</strong> ${resourceId}</p>
                    
                    <div class="alert alert-info">
                        <strong>What will happen:</strong>
                        <ul>
                            <li>The existing resource will be imported into your Terraform state</li>
                            <li>You'll need to add the corresponding resource block to your Terraform configuration</li>
                            <li>Terraform will start managing this resource going forward</li>
                        </ul>
                    </div>
                </div>
            </div>
        `;
    }

    // Update navigation buttons
    updateButtons() {
        const prevBtn = document.getElementById('prevStepBtn');
        const nextBtn = document.getElementById('nextStepBtn');
        const importBtn = document.getElementById('importBtn');

        prevBtn.disabled = this.currentStep === 1;
        
        if (this.currentStep === this.totalSteps) {
            nextBtn.style.display = 'none';
            importBtn.style.display = 'inline-block';
        } else {
            nextBtn.style.display = 'inline-block';
            importBtn.style.display = 'none';
        }
    }

    // Go to previous step
    previousStep() {
        if (this.currentStep > 1) {
            this.currentStep--;
            this.showStep(this.currentStep);
        }
    }

    // Go to next step
    nextStep() {
        if (this.currentStep < this.totalSteps) {
            this.currentStep++;
            this.showStep(this.currentStep);
        }
    }

    // Execute the import
    async executeImport() {
        const resourceType = document.getElementById('resourceType').value;
        const resourceName = document.getElementById('resourceName').value;
        const resourceId = document.getElementById('resourceId').value;

        const importData = {
            type: resourceType,
            name: resourceName,
            id: resourceId
        };

        // Add provider-specific data
        if (resourceType.startsWith('aws_')) {
            importData.region = document.getElementById('awsRegion').value;
        } else if (resourceType.startsWith('google_')) {
            importData.project = document.getElementById('gcpProject').value;
            importData.zone = document.getElementById('gcpZone').value;
        } else if (resourceType.startsWith('azurerm_')) {
            importData.resource_group = document.getElementById('azureResourceGroup').value;
        }

        try {
            const response = await fetch('/api/v1/state/import', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(importData)
            });

            if (response.ok) {
                this.showAlert('Resource imported successfully!', 'success');
                $(this.modal).modal('hide');
                // Refresh the app state
                if (this.app && this.app.loadState) {
                    this.app.loadState();
                }
            } else {
                const error = await response.json();
                this.showAlert(`Import failed: ${error.message}`, 'danger');
            }
        } catch (error) {
            this.showAlert(`Import failed: ${error.message}`, 'danger');
        }
    }

    // Show alert message
    showAlert(message, type) {
        const alertHTML = `
            <div class="alert alert-${type} alert-dismissible fade show" role="alert">
                ${message}
                <button type="button" class="close" data-dismiss="alert">
                    <span>&times;</span>
                </button>
            </div>
        `;
        
        const modalBody = this.modal.querySelector('.modal-body');
        modalBody.insertAdjacentHTML('afterbegin', alertHTML);
        
        // Auto-dismiss after 5 seconds
        setTimeout(() => {
            const alert = modalBody.querySelector('.alert');
            if (alert) {
                alert.remove();
            }
        }, 5000);
    }

    // Open the wizard
    open() {
        this.currentStep = 1;
        this.showStep(1);
        $(this.modal).modal('show');
    }
}

// Export the UI enhancement classes
window.BackendConfigurationUI = BackendConfigurationUI;
window.ResourceMoveUI = ResourceMoveUI;
window.ResourceRemovalUI = ResourceRemovalUI;
window.ImportWizardUI = ImportWizardUI;
