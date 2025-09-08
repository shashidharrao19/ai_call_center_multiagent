/**
 * WebSocket client for communication with the Go backend
 */

export class WebSocketClient {
    constructor() {
        this.ws = null;
        this.url = this.getWebSocketURL();
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        
        this.onMessage = null;
        this.onError = null;
        this.onClose = null;
        this.onOpen = null;
    }
    
    getWebSocketURL() {
        // Get WebSocket URL from environment or use default
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.hostname;
        const port = import.meta.env.VITE_WS_PORT || '8080';
        const path = import.meta.env.VITE_WS_PATH || '/ws';
        
        return `${protocol}//${host}:${port}${path}`;
    }
    
    async connect() {
        return new Promise((resolve, reject) => {
            try {
                console.log('Connecting to WebSocket:', this.url);
                
                this.ws = new WebSocket(this.url);
                
                this.ws.onopen = (event) => {
                    console.log('WebSocket connected');
                    this.reconnectAttempts = 0;
                    
                    if (this.onOpen) {
                        this.onOpen(event);
                    }
                    
                    resolve();
                };
                
                this.ws.onmessage = (event) => {
                    try {
                        const message = JSON.parse(event.data);
                        if (this.onMessage) {
                            this.onMessage(message);
                        }
                    } catch (error) {
                        console.error('Failed to parse WebSocket message:', error);
                    }
                };
                
                this.ws.onerror = (error) => {
                    console.error('WebSocket error:', error);
                    if (this.onError) {
                        this.onError(error);
                    }
                    reject(error);
                };
                
                this.ws.onclose = (event) => {
                    console.log('WebSocket closed:', event.code, event.reason);
                    
                    if (this.onClose) {
                        this.onClose(event);
                    }
                    
                    // Attempt to reconnect if not a normal closure
                    if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
                        this.attemptReconnect();
                    }
                };
                
            } catch (error) {
                console.error('Failed to create WebSocket connection:', error);
                reject(error);
            }
        });
    }
    
    attemptReconnect() {
        this.reconnectAttempts++;
        console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
        
        setTimeout(() => {
            this.connect().catch(error => {
                console.error('Reconnection failed:', error);
            });
        }, this.reconnectDelay * this.reconnectAttempts);
    }
    
    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            try {
                const data = JSON.stringify(message);
                this.ws.send(data);
                console.log('Sent message:', message);
            } catch (error) {
                console.error('Failed to send WebSocket message:', error);
                throw error;
            }
        } else {
            console.warn('WebSocket is not connected');
            throw new Error('WebSocket is not connected');
        }
    }
    
    isConnected() {
        return this.ws && this.ws.readyState === WebSocket.OPEN;
    }
    
    disconnect() {
        if (this.ws) {
            this.ws.close(1000, 'Client disconnecting');
            this.ws = null;
        }
    }
    
    // Send binary data (for audio)
    sendBinary(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(data);
        } else {
            throw new Error('WebSocket is not connected');
        }
    }
}
