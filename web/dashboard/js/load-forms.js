// Load Forms HTML Content
document.addEventListener('DOMContentLoaded', function() {
    loadFormHTML();
});

async function loadFormHTML() {
    try {
        // Load backend forms
        const backendFormsResponse = await fetch('forms/backend-form.html');
        const backendFormsHTML = await backendFormsResponse.text();
        document.getElementById('backend-forms').innerHTML = backendFormsHTML;

        // Load state forms
        const stateFormsResponse = await fetch('forms/state-form.html');
        const stateFormsHTML = await stateFormsResponse.text();
        document.getElementById('state-forms').innerHTML = stateFormsHTML;

        // Load remediation forms
        const remediationFormsResponse = await fetch('forms/remediation-form.html');
        const remediationFormsHTML = await remediationFormsResponse.text();
        document.getElementById('remediation-forms').innerHTML = remediationFormsHTML;

        console.log('Forms loaded successfully');
    } catch (error) {
        console.error('Error loading forms:', error);
        // Fallback: create basic modals if files can't be loaded
        createFallbackModals();
    }
}

function createFallbackModals() {
    // Create basic modals as fallback
    const backendForms = document.getElementById('backend-forms');
    backendForms.innerHTML = `
        <div id="backend-form-modal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2>Backend Form</h2>
                    <button class="modal-close" onclick="closeBackendForm()">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <div class="modal-body">
                    <p>Backend form content would be loaded here.</p>
                </div>
            </div>
        </div>
    `;

    const stateForms = document.getElementById('state-forms');
    stateForms.innerHTML = `
        <div id="state-form-modal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2>State Form</h2>
                    <button class="modal-close" onclick="closeStateForm()">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <div class="modal-body">
                    <p>State form content would be loaded here.</p>
                </div>
            </div>
        </div>
    `;

    const remediationForms = document.getElementById('remediation-forms');
    remediationForms.innerHTML = `
        <div id="remediation-form-modal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h2>Remediation Form</h2>
                    <button class="modal-close" onclick="closeRemediationForm()">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <div class="modal-body">
                    <p>Remediation form content would be loaded here.</p>
                </div>
            </div>
        </div>
    `;
}
