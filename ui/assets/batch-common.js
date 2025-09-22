// Common JavaScript functions for batch pages
// Note: This file depends on api-common.js for API call functionality

// Base batch app functionality
function createBatchApp(config) {
    return {
        items: [],
        loading: false,
        error: '',
        refreshInterval: null,
        lastRefresh: null,

        async loadItems() {
            console.log(`Loading ${config.itemName}...`);
            this.loading = true;
            this.error = '';
            
            try {
                const response = await apiCall('GET', config.apiEndpoint);
                if (response.data.success) {
                    this.items = response.data.data;
                    console.log(`Loaded ${this.items.length} ${config.itemName}`);
                } else {
                    this.error = response.data.error;
                    console.error('API error:', response.data.error);
                }
            } catch (err) {
                this.error = err.response?.data?.error || `Failed to load ${config.itemName}`;
                console.error('Request error:', err);
            } finally {
                this.loading = false;
                this.lastRefresh = new Date();
            }
        },

        formatDateTime(dateString) {
            const date = new Date(dateString);
            return date.toLocaleString('en-US', {
                timeZone: 'Asia/Jakarta',
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit',
                hour12: false
            });
        },

        startAutoRefresh() {
            console.log('Starting auto-refresh...');
            this.refreshInterval = setInterval(() => {
                console.log(`Auto-refreshing ${config.itemName}...`);
                this.loadItems();
            }, 10000);
            console.log('Auto-refresh started, interval ID:', this.refreshInterval);
        },

        stopAutoRefresh() {
            if (this.refreshInterval) {
                clearInterval(this.refreshInterval);
                this.refreshInterval = null;
            }
        },

        async clearAllItems() {
            if (!confirm(`Are you sure you want to clear ALL items from the ${config.queueName}? This action cannot be undone.`)) {
                return;
            }
            
            try {
                const response = await apiCall('DELETE', config.clearAllEndpoint);
                if (response.data.success) {
                    this.loadItems(); // Refresh the list
                } else {
                    this.error = response.data.error || 'Failed to clear all items';
                }
            } catch (err) {
                this.error = err.response?.data?.error || 'Failed to clear all items';
                console.error('Clear all error:', err);
            }
        },

        logout() {
            redirectToLogin();
        }
    };
}

function setupBatchPageCleanup() {
    window.addEventListener('beforeunload', () => {
        // Stop any running intervals
        const app = Alpine.$data(document.querySelector('[x-data]'));
        if (app && app.stopAutoRefresh) {
            app.stopAutoRefresh();
        }
    });
}