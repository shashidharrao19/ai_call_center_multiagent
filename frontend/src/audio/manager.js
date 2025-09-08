/**
 * Audio Manager - Handles audio capture, processing, and playback
 * Based on the notebook implementation but adapted for browser use
 */

export class AudioManager {
    constructor() {
        this.audioContext = null;
        this.mediaStream = null;
        this.processor = null;
        this.isCapturing = false;
        
        // Audio configuration
        this.sampleRate = 24000;
        this.channels = 1;
        this.chunkSize = 1024;
        
        // Callbacks
        this.onAudioData = null;
        this.onPlaybackData = null;
        
        // Audio queue for playback
        this.audioQueue = [];
        this.isPlaying = false;
    }
    
    async initialize() {
        try {
            // Create audio context
            this.audioContext = new (window.AudioContext || window.webkitAudioContext)({
                sampleRate: this.sampleRate
            });
            
            // Resume audio context if suspended
            if (this.audioContext.state === 'suspended') {
                await this.audioContext.resume();
            }
            
            console.log('Audio context initialized:', this.audioContext.sampleRate);
            
        } catch (error) {
            console.error('Failed to initialize audio context:', error);
            throw error;
        }
    }
    
    async startCapture() {
        try {
            if (this.isCapturing) {
                console.warn('Audio capture already started');
                return;
            }
            
            // Request microphone access
            this.mediaStream = await navigator.mediaDevices.getUserMedia({
                audio: {
                    sampleRate: this.sampleRate,
                    channelCount: this.channels,
                    echoCancellation: true,
                    noiseSuppression: true,
                    autoGainControl: true
                }
            });
            
            // Create media stream source
            const source = this.audioContext.createMediaStreamSource(this.mediaStream);
            
            // Create script processor for audio processing
            this.processor = this.audioContext.createScriptProcessor(this.chunkSize, this.channels, this.channels);
            
            // Setup audio processing
            this.processor.onaudioprocess = (event) => {
                this.processAudioInput(event);
            };
            
            // Connect audio nodes
            source.connect(this.processor);
            this.processor.connect(this.audioContext.destination);
            
            this.isCapturing = true;
            console.log('Audio capture started');
            
        } catch (error) {
            console.error('Failed to start audio capture:', error);
            throw error;
        }
    }
    
    stopCapture() {
        try {
            if (!this.isCapturing) {
                return;
            }
            
            // Stop media stream
            if (this.mediaStream) {
                this.mediaStream.getTracks().forEach(track => track.stop());
                this.mediaStream = null;
            }
            
            // Disconnect processor
            if (this.processor) {
                this.processor.disconnect();
                this.processor = null;
            }
            
            this.isCapturing = false;
            console.log('Audio capture stopped');
            
        } catch (error) {
            console.error('Error stopping audio capture:', error);
        }
    }
    
    processAudioInput(event) {
        try {
            // Get audio data from input
            const inputBuffer = event.inputBuffer;
            const inputData = inputBuffer.getChannelData(0);
            
            // Convert float32 to int16 PCM
            const pcmData = this.float32ToInt16(inputData);
            
            // Send audio data to callback
            if (this.onAudioData) {
                this.onAudioData(pcmData);
            }
            
        } catch (error) {
            console.error('Error processing audio input:', error);
        }
    }
    
    float32ToInt16(float32Array) {
        const int16Array = new Int16Array(float32Array.length);
        for (let i = 0; i < float32Array.length; i++) {
            // Convert from [-1, 1] to [-32768, 32767]
            const sample = Math.max(-1, Math.min(1, float32Array[i]));
            int16Array[i] = sample < 0 ? sample * 0x8000 : sample * 0x7FFF;
        }
        return int16Array;
    }
    
    int16ToFloat32(int16Array) {
        const float32Array = new Float32Array(int16Array.length);
        for (let i = 0; i < int16Array.length; i++) {
            // Convert from [-32768, 32767] to [-1, 1]
            float32Array[i] = int16Array[i] / (int16Array[i] < 0 ? 0x8000 : 0x7FFF);
        }
        return float32Array;
    }
    
    playAudio(audioData) {
        try {
            // Add audio data to queue
            this.audioQueue.push(audioData);
            
            // Start playback if not already playing
            if (!this.isPlaying) {
                this.processAudioQueue();
            }
            
        } catch (error) {
            console.error('Error playing audio:', error);
        }
    }
    
    async processAudioQueue() {
        if (this.audioQueue.length === 0) {
            this.isPlaying = false;
            return;
        }
        
        this.isPlaying = true;
        
        try {
            // Get next audio data from queue
            const audioData = this.audioQueue.shift();
            
            // Convert to audio buffer
            const audioBuffer = await this.createAudioBuffer(audioData);
            
            // Create audio source
            const source = this.audioContext.createBufferSource();
            source.buffer = audioBuffer;
            source.connect(this.audioContext.destination);
            
            // Play audio
            source.start();
            
            // Process next audio when current finishes
            source.onended = () => {
                this.processAudioQueue();
            };
            
        } catch (error) {
            console.error('Error processing audio queue:', error);
            this.isPlaying = false;
        }
    }
    
    async createAudioBuffer(audioData) {
        try {
            // Create audio buffer
            const audioBuffer = this.audioContext.createBuffer(
                this.channels,
                audioData.length,
                this.sampleRate
            );
            
            // Fill buffer with audio data
            const channelData = audioBuffer.getChannelData(0);
            const floatData = this.int16ToFloat32(audioData);
            
            for (let i = 0; i < floatData.length; i++) {
                channelData[i] = floatData[i];
            }
            
            return audioBuffer;
            
        } catch (error) {
            console.error('Error creating audio buffer:', error);
            throw error;
        }
    }
    
    // Utility methods for audio format conversion
    encodeAudioToBase64(audioData) {
        const buffer = audioData.buffer.slice(
            audioData.byteOffset,
            audioData.byteOffset + audioData.byteLength
        );
        return btoa(String.fromCharCode(...new Uint8Array(buffer)));
    }
    
    decodeAudioFromBase64(base64String) {
        const binaryString = atob(base64String);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }
        return new Int16Array(bytes.buffer);
    }
    
    // Get audio level for visualization
    getAudioLevel() {
        if (!this.processor || !this.isCapturing) {
            return 0;
        }
        
        // This would need to be implemented with an analyzer node
        // For now, return a random value for visualization
        return Math.random() * 0.5 + 0.3;
    }
    
    // Cleanup
    cleanup() {
        this.stopCapture();
        
        if (this.audioContext) {
            this.audioContext.close();
            this.audioContext = null;
        }
        
        this.audioQueue = [];
        this.isPlaying = false;
    }
}
