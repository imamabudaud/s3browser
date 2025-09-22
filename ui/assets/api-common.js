async function apiCall(method, url, data = null) {
    const token = localStorage.getItem('token');
    
    if (!token && url !== '/api/login') {
        redirectToLogin();
        return Promise.reject(new Error('No authentication token'));
    }
    
    const config = {
        method,
        url,
        headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
        }
    };
    
    if (data) {
        config.data = data;
    }
    
    try {
        return await axios(config);
    } catch (error) {
        if (error.response && error.response.status === 401) {
            console.log('Token expired, redirecting to login...');
            redirectToLogin();
            return Promise.reject(new Error('Authentication token expired'));
        }
        
        throw error;
    }
}

function redirectToLogin() {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    window.location.href = '/login';
}

function isAuthenticated() {
    const token = localStorage.getItem('token');
    return !!token;
}

function getCurrentUser() {
    const userStr = localStorage.getItem('user');
    if (userStr) {
        try {
            return JSON.parse(userStr);
        } catch (e) {
            console.error('Error parsing user data:', e);
            return null;
        }
    }
    return null;
}

async function uploadFormData(url, formData) {
    const token = localStorage.getItem('token');
    
    if (!token) {
        redirectToLogin();
        return Promise.reject(new Error('No authentication token'));
    }
    
    try {
        return await axios.post(url, formData, {
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'multipart/form-data'
            }
        });
    } catch (error) {
        if (error.response && error.response.status === 401) {
            console.log('Token expired, redirecting to login...');
            redirectToLogin();
            return Promise.reject(new Error('Authentication token expired'));
        }
        
        throw error;
    }
}

function logout() {
    redirectToLogin();
}
