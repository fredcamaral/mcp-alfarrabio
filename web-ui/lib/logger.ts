/**
 * Production-ready logging system for the web UI
 * Provides structured logging with different levels and environment awareness
 */

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface LogContext {
  component?: string;
  action?: string;
  userId?: string;
  sessionId?: string;
  correlationId?: string;
  [key: string]: string | number | boolean | undefined;
}

export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  context?: LogContext;
  error?: {
    name: string;
    message: string;
    stack?: string;
  };
}

class Logger {
  private isDevelopment = process.env.NODE_ENV === 'development';
  private logLevel: LogLevel = (process.env.NEXT_PUBLIC_LOG_LEVEL as LogLevel) || 'info';
  private logBuffer: LogEntry[] = [];
  private maxBufferSize = 100;

  private logLevels: Record<LogLevel, number> = {
    debug: 0,
    info: 1,
    warn: 2,
    error: 3,
  };

  private shouldLog(level: LogLevel): boolean {
    return this.logLevels[level] >= this.logLevels[this.logLevel];
  }

  private formatLogEntry(level: LogLevel, message: string, context?: LogContext, error?: Error): LogEntry {
    return {
      timestamp: new Date().toISOString(),
      level,
      message,
      context,
      error: error ? {
        name: error.name,
        message: error.message,
        stack: this.isDevelopment ? error.stack : undefined,
      } : undefined,
    };
  }

  private sendToLoggingService(entry: LogEntry): void {
    // In production, this would send to a logging service like DataDog, Sentry, etc.
    if (!this.isDevelopment && typeof window !== 'undefined') {
      // Buffer logs to send in batches
      this.logBuffer.push(entry);
      
      if (this.logBuffer.length >= this.maxBufferSize) {
        this.flushLogs();
      }
    }
  }

  private flushLogs(): void {
    if (this.logBuffer.length === 0) return;

    // In production, implement batch sending to logging service
    const endpoint = process.env.NEXT_PUBLIC_LOGGING_ENDPOINT;
    if (endpoint) {
      fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ logs: this.logBuffer }),
      }).catch(() => {
        // Silently fail to avoid infinite error loops
      });
    }

    this.logBuffer = [];
  }

  private log(level: LogLevel, message: string, context?: LogContext, error?: Error): void {
    if (!this.shouldLog(level)) return;

    const entry = this.formatLogEntry(level, message, context, error);

    // Development: Use console with color coding
    if (this.isDevelopment) {
      const styles = {
        debug: 'color: #888',
        info: 'color: #0066cc',
        warn: 'color: #ff9800',
        error: 'color: #f44336',
      };

      const consoleMethod = level === 'error' ? 'error' : level === 'warn' ? 'warn' : 'log';
      
      console[consoleMethod](
        `%c[${entry.timestamp}] [${level.toUpperCase()}] ${message}`,
        styles[level],
        context || '',
        error || ''
      );
    }

    // Production: Send to logging service
    this.sendToLoggingService(entry);
  }

  debug(message: string, context?: LogContext): void {
    this.log('debug', message, context);
  }

  info(message: string, context?: LogContext): void {
    this.log('info', message, context);
  }

  warn(message: string, context?: LogContext): void {
    this.log('warn', message, context);
  }

  error(message: string, error?: Error | unknown, context?: LogContext): void {
    const errorObj = error instanceof Error ? error : new Error(String(error));
    this.log('error', message, context, errorObj);
  }

  // Utility method for performance tracking
  time(label: string, context?: LogContext): () => void {
    const start = performance.now();
    return () => {
      const duration = performance.now() - start;
      this.debug(`${label} completed`, {
        ...context,
        duration: `${duration.toFixed(2)}ms`,
      });
    };
  }

  // Ensure logs are sent when page unloads
  setupUnloadHandler(): void {
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', () => {
        this.flushLogs();
      });
    }
  }
}

// Export singleton instance
export const logger = new Logger();

// Setup unload handler on client side
if (typeof window !== 'undefined') {
  logger.setupUnloadHandler();
}