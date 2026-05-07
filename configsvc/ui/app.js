document.addEventListener('DOMContentLoaded', () => {
    const API_BASE = 'http://localhost:8089/api/v1/configs';

    const serviceList = document.getElementById('service-list');
    const fileNameDisplay = document.getElementById('current-file-name');
    
    const selectors = document.getElementById('selectors');
    const envSelect = document.getElementById('env-select');
    const templateSelect = document.getElementById('template-select');
    
    const resultEditor = document.getElementById('result-editor');
    const valueEditor = document.getElementById('value-editor');
    const templateEditor = document.getElementById('template-editor');
    
    const btnFormat = document.getElementById('btn-format');
    const btnSaveUpdate = document.getElementById('btn-save-update');
    const btnAddService = document.getElementById('btn-add-service');
    
    const saveStatus = document.getElementById('save-status');
    const errorBar = document.getElementById('error-bar');
    const errorText = document.getElementById('error-text');

    let currentService = null;
    let currentTemplateName = null;
    let currentOriginalTemplateStr = "";
    
    // Manage environment states
    let valuesStore = {};
    let currentEnv = "base";

    async function fetchAPI(endpoint, options = {}) {
        try {
            const res = await fetch(endpoint, options);
            if (!res.ok) throw new Error(`API Error: ${res.status}`);
            const isJson = res.headers.get('content-type')?.includes('application/json');
            return isJson ? await res.json() : await res.text();
        } catch (err) {
            console.error(err);
            showError("Network Error: Could not connect to API");
            throw err;
        }
    }

    async function loadSidebar() {
        serviceList.innerHTML = '<li><span style="padding:10px; color:#a1a1aa">Loading...</span></li>';
        const services = await fetchAPI(API_BASE);
        
        serviceList.innerHTML = '';
        services.forEach(svc => {
            const li = document.createElement('li');
            li.className = 'tree-item';
            li.innerHTML = `<span class="material-symbols-outlined">data_object</span> ${svc}`;
            li.addEventListener('click', () => selectService(svc, li));
            serviceList.appendChild(li);
        });
    }

    async function selectService(serviceName, element) {
        document.querySelectorAll('.tree-item').forEach(el => el.classList.remove('active'));
        if (element) element.classList.add('active');

        currentService = serviceName;
        fileNameDisplay.textContent = `Configs / ${serviceName}`;
        selectors.style.display = 'flex';
        
        // Fetch versions from API
        const versions = await fetchAPI(`${API_BASE}/${serviceName}/versions`);
        
        templateSelect.innerHTML = '';
        if (versions && Array.isArray(versions)) {
            versions.forEach(vInfo => {
                const versionVal = typeof vInfo === 'string' ? vInfo : vInfo.version;
                const opt = document.createElement('option');
                opt.value = versionVal;
                opt.textContent = `${versionVal}.jsonnet`;
                templateSelect.appendChild(opt);
            });
        }
        
        if (versions && versions.length > 0) {
            const firstVal = typeof versions[0] === 'string' ? versions[0] : versions[0].version;
            loadTemplateAndValues(firstVal);
        } else {
            const defaultVersion = "v1.0.0";
            const opt = document.createElement('option');
            opt.value = defaultVersion;
            opt.textContent = `${defaultVersion}.jsonnet`;
            templateSelect.appendChild(opt);
            templateSelect.value = defaultVersion;
            currentTemplateName = defaultVersion;

            templateEditor.value = "{\n  \n}";
            valuesStore = {
                base: "{\n  \n}",
                dev: "{\n  \n}",
                staging: "{\n  \n}",
                prod: "{\n  \n}"
            };
            envSelect.value = "base";
            currentEnv = "base";
            valueEditor.value = valuesStore["base"];
            btnSaveUpdate.disabled = false;
        }
    }

    // --- New Service Logic ---
    btnAddService.addEventListener('click', () => {
        const newSvc = prompt("Enter new service name (e.g. email-service):");
        if (!newSvc) return;
        
        currentService = newSvc.trim();
        const v = "v1.0.0";
        
        fileNameDisplay.textContent = `Configs / ${currentService}`;
        selectors.style.display = 'flex';
        
        templateSelect.innerHTML = `<option value="${v}">${v}.jsonnet</option>`;
        templateSelect.value = v;
        currentTemplateName = v;
        
        // Clear active tree item
        document.querySelectorAll('.tree-item').forEach(el => el.classList.remove('active'));
        
        templateEditor.value = "{\n  \n}";
        valuesStore = {
            base: "{\n  \n}",
            dev: "{\n  \n}",
            staging: "{\n  \n}",
            prod: "{\n  \n}"
        };
        envSelect.value = "base";
        currentEnv = "base";
        valueEditor.value = valuesStore["base"];
        
        btnSaveUpdate.disabled = false;
        saveStatus.className = 'status-badge visible unsaved';
        saveStatus.textContent = 'Draft (Unsaved)';
        
        debouncedUpdateResult();
    });

    async function loadTemplateAndValues(templateName) {
        currentTemplateName = templateName;
        templateSelect.value = templateName;
        
        try {
            const data = await fetchAPI(`${API_BASE}/${currentService}/bundle/${encodeURIComponent(templateName)}`);
            templateEditor.value = data.template;
            
            // Populate values store
            valuesStore = {
                base: data.values["base"] || "{\n  \n}",
                dev: data.values["dev"] || "{\n  \n}",
                staging: data.values["staging"] || "{\n  \n}",
                prod: data.values["prod"] || "{\n  \n}"
            };
        } catch(e) {
            templateEditor.value = "{\n  \n}";
            valuesStore = {
                base: "{\n  \n}",
                dev: "{\n  \n}",
                staging: "{\n  \n}",
                prod: "{\n  \n}"
            };
        }
        
        currentEnv = "base";
        envSelect.value = currentEnv;
        valueEditor.value = valuesStore[currentEnv];
        
        currentOriginalTemplateStr = templateEditor.value;
        
        btnSaveUpdate.disabled = false;
        hideError();
        checkUnsavedChanges();
        debouncedUpdateResult();
    }

    templateSelect.addEventListener('change', (e) => {
        if (!currentService) return;
        if (templateEditor.value !== currentOriginalTemplateStr) {
            if(!confirm("You have unsaved template changes. Discard?")) {
                e.target.value = currentTemplateName; // revert
                return;
            }
        }
        loadTemplateAndValues(e.target.value);
    });
    
    envSelect.addEventListener('change', (e) => {
        if (!currentService) return;
        // Save current editor to old env
        valuesStore[currentEnv] = valueEditor.value;
        // Load new env
        currentEnv = e.target.value;
        valueEditor.value = valuesStore[currentEnv] || "{\n  \n}";
        debouncedUpdateResult();
    });

    // --- Editor Events ---
    templateEditor.addEventListener('input', () => {
        checkUnsavedChanges();
        debouncedUpdateResult();
    });
    
    valueEditor.addEventListener('input', () => {
        valuesStore[currentEnv] = valueEditor.value;
        checkUnsavedChanges();
        debouncedUpdateResult();
    });

    function checkUnsavedChanges() {
        if (!currentService) return;
        // simplified check
        if (templateEditor.value !== currentOriginalTemplateStr) {
            saveStatus.textContent = 'Unsaved changes';
            saveStatus.className = 'status-badge visible unsaved';
        } else {
            saveStatus.className = 'status-badge';
        }
    }

    function hideError() { errorBar.classList.remove('visible'); }
    function showError(msg) {
        errorText.textContent = msg;
        errorBar.classList.add('visible');
    }

    let debounceTimer;
    function debouncedUpdateResult() {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(updateResult, 500);
    }

    async function updateResult() {
        const tmpl = templateEditor.value;
        const baseVals = valuesStore["base"] || "{}";
        const envVals = (currentEnv === "base") ? "{}" : (valuesStore[currentEnv] || "{}");
        
        if (tmpl.trim() === '') {
            resultEditor.value = "";
            hideError();
            return;
        }

        try {
            const res = await fetch(`${API_BASE}/render`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ 
                    template: tmpl, 
                    base_values: baseVals,
                    env_values: envVals
                })
            });
            
            const data = await res.json();
            
            if (!res.ok) {
                // If it's a jsonnet evaluation error, we show it in the result box or error bar
                resultEditor.value = `// Rendering Error:\n${data.error || 'Unknown error'}`;
                showError("Jsonnet Evaluation Error");
                return;
            }

            resultEditor.value = JSON.stringify(data, null, 2);
            hideError();
        } catch (e) {
            resultEditor.value = `// Network or server error:\n${e.message}`;
            showError("Failed to reach rendering server");
        }
    }

    btnFormat.addEventListener('click', () => {
        try {
            if (valueEditor.value.trim() !== "") {
                const parsedVal = JSON.parse(valueEditor.value);
                valueEditor.value = JSON.stringify(parsedVal, null, 2);
            }
            // we won't format Jsonnet for now since standard JSON.parse breaks on it
            hideError();
        } catch (e) {
            showError("Cannot format: Invalid JSON");
        }
    });

    btnSaveUpdate.addEventListener('click', async () => {
        if (!currentService) return;
        try {
            saveStatus.className = 'status-badge visible';
            saveStatus.textContent = "Updating files...";

            // Ensure current editor values are flushed to store
            valuesStore[currentEnv] = valueEditor.value;

            // Make PUT request to bundle endpoint
            const payload = {
                template: templateEditor.value,
                values: valuesStore
            };

            const res = await fetch(`${API_BASE}/${currentService}/bundle/${currentTemplateName}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!res.ok) throw new Error("Failed to save");

            currentOriginalTemplateStr = templateEditor.value;

            saveStatus.textContent = "Saved!";
            saveStatus.classList.remove('unsaved');
            setTimeout(() => saveStatus.className = 'status-badge', 3000);
            hideError();
        } catch (e) {
            showError("Failed to save to S3.");
        }
    });

    function handleTab(e) {
        if (e.key === 'Tab') {
            e.preventDefault();
            const start = this.selectionStart;
            const end = this.selectionEnd;
            this.value = this.value.substring(0, start) + "  " + this.value.substring(end);
            this.selectionStart = this.selectionEnd = start + 2;
        }
    }

    templateEditor.addEventListener('keydown', handleTab);
    valueEditor.addEventListener('keydown', handleTab);

    // Boot UI
    loadSidebar();
});
