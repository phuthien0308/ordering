document.addEventListener('DOMContentLoaded', () => {
    const API_BASE = 'http://localhost:8089/api/v1/configs';

    const serviceList = document.getElementById('service-list');
    const fileNameDisplay = document.getElementById('current-file-name');
    const versionSelect = document.getElementById('version-select');
    const editor = document.getElementById('json-editor');
    const lineNumbers = document.getElementById('line-numbers');
    
    const btnFormat = document.getElementById('btn-format');
    const btnSaveUpdate = document.getElementById('btn-save-update');
    const btnSaveNew = document.getElementById('btn-save-new');
    const btnAddService = document.getElementById('btn-add-service');
    
    const saveStatus = document.getElementById('save-status');
    const errorBar = document.getElementById('error-bar');
    const errorText = document.getElementById('error-text');

    let currentService = null;
    let currentVersion = null;
    let currentOriginalJSONStr = "";

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
            li.innerHTML = `<span class="material-symbols-outlined">data_object</span> ${svc}.json`;
            li.addEventListener('click', () => selectService(svc, li));
            serviceList.appendChild(li);
        });
    }

    async function selectService(serviceName, element) {
        document.querySelectorAll('.tree-item').forEach(el => el.classList.remove('active'));
        if (element) element.classList.add('active');

        currentService = serviceName;
        fileNameDisplay.textContent = `Configs / ${serviceName}.json`;
        
        // Fetch versions from API
        const versions = await fetchAPI(`${API_BASE}/${serviceName}/versions`);
        
        versionSelect.innerHTML = '';
        versions.forEach(vInfo => {
            const versionVal = typeof vInfo === 'string' ? vInfo : vInfo.version;
            const opt = document.createElement('option');
            opt.value = versionVal;
            opt.textContent = versionVal;
            versionSelect.appendChild(opt);
        });
        
        versionSelect.style.display = 'block';
        
        // Load the first available version (hopefully latest) if array has items
        if (versions.length > 0) {
            const firstVal = typeof versions[0] === 'string' ? versions[0] : versions[0].version;
            loadVersionData(firstVal);
        }
    }

    // --- New Service Logic ---
    btnAddService.addEventListener('click', () => {
        const newSvc = prompt("Enter new service name (e.g. email-service):");
        if (!newSvc) return;
        
        currentService = newSvc.trim();
        currentVersion = "v1.0.0 (Latest)";
        
        fileNameDisplay.textContent = `Configs / ${currentService}.json`;
        
        versionSelect.innerHTML = `<option value="${currentVersion}">${currentVersion}</option>`;
        versionSelect.value = currentVersion;
        versionSelect.style.display = 'block';
        
        // Clear active tree item
        document.querySelectorAll('.tree-item').forEach(el => el.classList.remove('active'));
        
        currentOriginalJSONStr = "{\n  \n}";
        editor.value = currentOriginalJSONStr;
        
        btnSaveUpdate.disabled = false;
        btnSaveNew.disabled = false;
        
        updateLineNumbers();
        saveStatus.className = 'status-badge visible unsaved';
        saveStatus.textContent = 'Draft (Unsaved)';
    });

    async function loadVersionData(versionName) {
        currentVersion = versionName;
        versionSelect.value = versionName;
        
        const data = await fetchAPI(`${API_BASE}/${currentService}/versions/${encodeURIComponent(versionName)}`);
        
        // Data might be string returned or JSON object, format it securely
        const pureObj = typeof data === 'string' ? JSON.parse(data) : data;
        currentOriginalJSONStr = JSON.stringify(pureObj, null, 2);
        
        editor.value = currentOriginalJSONStr;
        
        btnSaveUpdate.disabled = false;
        btnSaveNew.disabled = false;
        
        hideError();
        updateLineNumbers();
        checkUnsavedChanges();
    }

    versionSelect.addEventListener('change', (e) => {
        if (!currentService) return;
        const newVersion = e.target.value;
        if (editor.value !== currentOriginalJSONStr) {
            if(!confirm("You have unsaved changes. Discard and switch version?")) {
                e.target.value = currentVersion; // revert
                return;
            }
        }
        loadVersionData(newVersion);
    });

    // --- Line Number logic ---
    function updateLineNumbers() {
        const lines = editor.value.split('\n').length;
        let numbersHtml = '';
        for (let i = 1; i <= lines; i++) {
            numbersHtml += i + '<br>';
        }
        lineNumbers.innerHTML = numbersHtml;
    }

    editor.addEventListener('scroll', () => lineNumbers.scrollTop = editor.scrollTop);

    editor.addEventListener('input', () => {
        updateLineNumbers();
        checkUnsavedChanges();
        validateJsonBackground();
    });

    function checkUnsavedChanges() {
        if (!currentService) return;
        if (editor.value !== currentOriginalJSONStr) {
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

    function validateJsonBackground() {
        try {
            if (editor.value.trim() === '') return hideError();
            JSON.parse(editor.value);
            hideError();
        } catch (e) {
            showError("Invalid JSON syntax");
        }
    }

    btnFormat.addEventListener('click', () => {
        try {
            const parsed = JSON.parse(editor.value);
            editor.value = JSON.stringify(parsed, null, 2);
            updateLineNumbers();
            hideError();
            checkUnsavedChanges();
        } catch (e) {
            showError("Cannot format: Invalid JSON");
        }
    });

    // --- Saving logic (Sends to API) ---
    btnSaveUpdate.addEventListener('click', () => {
        saveData(currentVersion, "Updating version...");
    });

    btnSaveNew.addEventListener('click', () => {
        if (!currentService) return;
        let newVersionName = prompt("Enter new version label (e.g. v2.1.0):");
        if (!newVersionName) return; 
        
        newVersionName = newVersionName + " (Latest)";
        saveData(newVersionName, `Creating ${newVersionName}...`, true);
    });

    async function saveData(versionKey, messageText, isNew = false) {
        if (!currentService) return;
        try {
            const parsed = JSON.parse(editor.value);
            saveStatus.className = 'status-badge visible';
            saveStatus.textContent = messageText;

            // Make PUT request
            await fetchAPI(`${API_BASE}/${currentService}/versions/${encodeURIComponent(versionKey)}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(parsed)
            });

            currentOriginalJSONStr = JSON.stringify(parsed, null, 2);
            editor.value = currentOriginalJSONStr;
            updateLineNumbers();

            saveStatus.textContent = "Saved to Database!";
            saveStatus.classList.remove('unsaved');
            setTimeout(() => saveStatus.className = 'status-badge', 3000);
            hideError();
            
            if (isNew) {
                // Refresh backend layout
                const currentActive = document.querySelector('.tree-item.active');
                selectService(currentService, currentActive);
                currentVersion = versionKey;
            }
        } catch (e) {
            showError(e.message || "Failed to save: Invalid JSON");
        }
    }

    editor.addEventListener('keydown', function(e) {
        if (e.key === 'Tab') {
            e.preventDefault();
            const start = this.selectionStart;
            const end = this.selectionEnd;
            this.value = this.value.substring(0, start) + "  " + this.value.substring(end);
            this.selectionStart = this.selectionEnd = start + 2;
            updateLineNumbers();
        }
    });

    // Boot UI by requesting the backend listing
    loadSidebar();
});
