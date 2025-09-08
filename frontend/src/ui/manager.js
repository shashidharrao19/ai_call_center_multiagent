/**
 * UI Manager - Handles user interface updates and interactions
 */

export class UIManager {
    constructor() {
        this.elements = {
            status: document.getElementById('status'),
            errorMessage: document.getElementById('error-message'),
            startCallBtn: document.getElementById('start-call'),
            endCallBtn: document.getElementById('end-call'),
            sendTextBtn: document.getElementById('send-text'),
            sessionInfo: document.getElementById('session-info'),
            sessionId: document.getElementById('session-id'),
            sessionStatus: document.getElementById('session-status'),
            callDuration: document.getElementById('call-duration')
        };
        
        this.isInitialized = false;
    }
    
    init() {
        if (this.isInitialized) {
            return;
        }
        
        // Setup initial state
        this.updateStatus('connecting', 'Initializing...');
        this.hideError();
        this.hideCallControls();
        this.hideSessionInfo();
        
        this.isInitialized = true;
        console.log('UI Manager initialized');
    }
    
    updateStatus(type, message) {
        const statusElement = this.elements.status;
        
        // Remove existing status classes
        statusElement.classList.remove('connecting', 'connected', 'error');
        
        // Add new status class
        statusElement.classList.add(type);
        
        // Update message
        statusElement.textContent = message;
        
        console.log(`Status updated: ${type} - ${message}`);
    }
    
    showError(message) {
        const errorElement = this.elements.errorMessage;
        errorElement.textContent = message;
        errorElement.classList.remove('hidden');
        
        // Auto-hide error after 5 seconds
        setTimeout(() => {
            this.hideError();
        }, 5000);
        
        console.error('Error displayed:', message);
    }
    
    hideError() {
        this.elements.errorMessage.classList.add('hidden');
    }
    
    showCallControls() {
        this.elements.startCallBtn.classList.add('hidden');
        this.elements.endCallBtn.classList.remove('hidden');
        this.elements.sendTextBtn.classList.remove('hidden');
    }
    
    hideCallControls() {
        this.elements.startCallBtn.classList.remove('hidden');
        this.elements.endCallBtn.classList.add('hidden');
        this.elements.sendTextBtn.classList.add('hidden');
    }
    
    showSessionInfo() {
        this.elements.sessionInfo.classList.remove('hidden');
    }
    
    hideSessionInfo() {
        this.elements.sessionInfo.classList.add('hidden');
    }
    
    updateSessionInfo(sessionData) {
        if (sessionData.session_id) {
            this.elements.sessionId.textContent = sessionData.session_id;
        }
        
        if (sessionData.status) {
            this.elements.sessionStatus.textContent = sessionData.status;
        }
        
        this.showSessionInfo();
    }
    
    updateCallDuration(duration) {
        this.elements.callDuration.textContent = duration;
    }
    
    showTextMessage(text) {
        // Create a temporary notification for text messages
        const notification = document.createElement('div');
        notification.className = 'text-notification';
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: #007bff;
            color: white;
            padding: 1rem;
            border-radius: 10px;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
            max-width: 300px;
            z-index: 1000;
            animation: slideIn 0.3s ease-out;
        `;
        
        notification.innerHTML = `
            <div style="font-weight: bold; margin-bottom: 0.5rem;">AI Response:</div>
            <div>${text}</div>
        `;
        
        // Add animation styles
        const style = document.createElement('style');
        style.textContent = `
            @keyframes slideIn {
                from { transform: translateX(100%); opacity: 0; }
                to { transform: translateX(0); opacity: 1; }
            }
            @keyframes slideOut {
                from { transform: translateX(0); opacity: 1; }
                to { transform: translateX(100%); opacity: 0; }
            }
        `;
        document.head.appendChild(style);
        
        document.body.appendChild(notification);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            notification.style.animation = 'slideOut 0.3s ease-in';
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.parentNode.removeChild(notification);
                }
            }, 300);
        }, 5000);
    }
    
    // Button state management
    setButtonEnabled(buttonId, enabled) {
        const button = document.getElementById(buttonId);
        if (button) {
            button.disabled = !enabled;
        }
    }
    
    setButtonText(buttonId, text) {
        const button = document.getElementById(buttonId);
        if (button) {
            button.textContent = text;
        }
    }
    
    // Loading states
    showLoading(buttonId) {
        const button = document.getElementById(buttonId);
        if (button) {
            button.disabled = true;
            button.textContent = 'Loading...';
        }
    }
    
    hideLoading(buttonId, originalText) {
        const button = document.getElementById(buttonId);
        if (button) {
            button.disabled = false;
            button.textContent = originalText;
        }
    }
    
    // Audio visualization
    updateAudioVisualization(level) {
        const bars = document.querySelectorAll('.audio-bar');
        bars.forEach((bar, index) => {
            const height = Math.max(20, level * 60 * (1 + Math.sin(Date.now() * 0.01 + index) * 0.3));
            bar.style.height = `${height}px`;
        });
    }
    
    // Responsive design helpers
    isMobile() {
        return window.innerWidth <= 768;
    }
    
    adjustForMobile() {
        if (this.isMobile()) {
            // Adjust UI for mobile devices
            const container = document.querySelector('.container');
            if (container) {
                container.style.padding = '1rem';
                container.style.maxWidth = '95%';
            }
        }
    }
    
    // Cleanup
    cleanup() {
        // Remove any temporary elements
        const notifications = document.querySelectorAll('.text-notification');
        notifications.forEach(notification => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        });
        
        console.log('UI Manager cleaned up');
    }
}
