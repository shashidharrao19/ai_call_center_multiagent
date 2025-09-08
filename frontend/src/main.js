/**
 * Main entry point for the AI Call Center frontend
 */

import { WebSocketClient } from './websocket/client.js';
import { AudioManager } from './audio/manager.js';
import { UIManager } from './ui/manager.js';

class CallCenterApp {
    constructor() {
        this.wsClient = new WebSocketClient();
        this.audioManager = new AudioManager();
        this.uiManager = new UIManager();
        
        this.sessionId = null;
        this.isCallActive = false;
        this.callStartTime = null;
        this.callDurationInterval = null;
        
        this.init();
    }
    
    async init() {
        try {
            // Initialize UI
            this.uiManager.init();
            
            // Setup event listeners
            this.setupEventListeners();
            
            // Connect to WebSocket
            await this.wsClient.connect();
            
            // Setup WebSocket message handlers
            this.setupWebSocketHandlers();
            
            this.uiManager.updateStatus('connected', 'Connected to server');
            
        } catch (error) {
            console.error('Failed to initialize app:', error);
            this.uiManager.showError('Failed to initialize application');
        }
    }
    
    setupEventListeners() {
        // Start call button
        document.getElementById('start-call').addEventListener('click', () => {
            this.startCall();
        });
        
        // End call button
        document.getElementById('end-call').addEventListener('click', () => {
            this.endCall();
        });
        
        // Send text button
        document.getElementById('send-text').addEventListener('click', () => {
            this.sendTextMessage();
        });
    }
    
    setupWebSocketHandlers() {
        this.wsClient.onMessage = (message) => {
            this.handleWebSocketMessage(message);
        };
        
        this.wsClient.onError = (error) => {
            console.error('WebSocket error:', error);
            this.uiManager.showError('Connection error: ' + error.message);
        };
        
        this.wsClient.onClose = () => {
            this.uiManager.updateStatus('error', 'Connection lost');
            this.endCall();
        };
    }
    
    async startCall() {
        try {
            this.uiManager.updateStatus('connecting', 'Starting call...');
            
            // Request microphone access
            await this.audioManager.initialize();
            
            // Start audio capture
            await this.audioManager.startCapture();
            
            // Setup audio data handler
            this.audioManager.onAudioData = (audioData) => {
                this.sendAudioData(audioData);
            };
            
            // Setup audio playback handler
            this.audioManager.onPlaybackData = (audioData) => {
                this.audioManager.playAudio(audioData);
            };
            
            this.isCallActive = true;
            this.callStartTime = Date.now();
            this.startCallDurationTimer();
            
            this.uiManager.updateStatus('connected', 'Call active');
            this.uiManager.showCallControls();
            
            console.log('Call started successfully');
            
        } catch (error) {
            console.error('Failed to start call:', error);
            this.uiManager.showError('Failed to start call: ' + error.message);
            this.uiManager.updateStatus('error', 'Call failed');
        }
    }
    
    endCall() {
        try {
            // Stop audio capture
            this.audioManager.stopCapture();
            
            // Stop call duration timer
            if (this.callDurationInterval) {
                clearInterval(this.callDurationInterval);
                this.callDurationInterval = null;
            }
            
            // Send end call message
            if (this.isCallActive) {
                this.wsClient.send({
                    type: 'control',
                    data: {
                        action: 'end_call'
                    }
                });
            }
            
            this.isCallActive = false;
            this.callStartTime = null;
            
            this.uiManager.hideCallControls();
            this.uiManager.updateStatus('connected', 'Ready for new call');
            
            console.log('Call ended');
            
        } catch (error) {
            console.error('Error ending call:', error);
        }
    }
    
    sendTextMessage() {
        const text = prompt('Enter your message:');
        if (text && text.trim()) {
            this.wsClient.send({
                type: 'text',
                data: {
                    text: text.trim()
                }
            });
        }
    }
    
    sendAudioData(audioData) {
        if (this.isCallActive && this.wsClient.isConnected()) {
            this.wsClient.send({
                type: 'audio',
                data: {
                    data: audioData,
                    format: 'PCM',
                    timestamp: Date.now()
                }
            });
        }
    }
    
    handleWebSocketMessage(message) {
        try {
            console.log('Received message:', message);
            
            switch (message.type) {
                case 'session':
                    this.handleSessionMessage(message);
                    break;
                case 'audio':
                    this.handleAudioMessage(message);
                    break;
                case 'text':
                    this.handleTextMessage(message);
                    break;
                case 'error':
                    this.handleErrorMessage(message);
                    break;
                case 'heartbeat':
                    // Handle heartbeat - no action needed
                    break;
                default:
                    console.warn('Unknown message type:', message.type);
            }
            
        } catch (error) {
            console.error('Error handling WebSocket message:', error);
        }
    }
    
    handleSessionMessage(message) {
        if (message.data && message.data.session_id) {
            this.sessionId = message.data.session_id;
            this.uiManager.updateSessionInfo(message.data);
        }
    }
    
    handleAudioMessage(message) {
        if (message.data && message.data.data) {
            // Play received audio
            this.audioManager.playAudio(message.data.data);
        }
    }
    
    handleTextMessage(message) {
        if (message.data && message.data.text) {
            // Display text message (could be AI response or function call result)
            this.uiManager.showTextMessage(message.data.text);
        }
    }
    
    handleErrorMessage(message) {
        if (message.data && message.data.error) {
            this.uiManager.showError(message.data.error);
        }
    }
    
    startCallDurationTimer() {
        this.callDurationInterval = setInterval(() => {
            if (this.callStartTime) {
                const duration = Date.now() - this.callStartTime;
                const minutes = Math.floor(duration / 60000);
                const seconds = Math.floor((duration % 60000) / 1000);
                const timeString = `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
                this.uiManager.updateCallDuration(timeString);
            }
        }, 1000);
    }
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new CallCenterApp();
});
