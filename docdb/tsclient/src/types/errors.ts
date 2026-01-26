import { Status } from './protocol';

export class DocDBError extends Error {
  code: Status;

  constructor(message: string, code: Status) {
    super(message);
    this.name = 'DocDBError';
    this.code = code;
  }
}

export class ConnectionError extends DocDBError {
  constructor(message: string = 'Failed to connect to server') {
    super(message, Status.Error);
    this.name = 'ConnectionError';
  }
}

export class ValidationError extends DocDBError {
  constructor(message: string) {
    super(message, Status.Error);
    this.name = 'ValidationError';
  }
}

export class TimeoutError extends DocDBError {
  constructor(message: string = 'Operation timed out') {
    super(message, Status.Error);
    this.name = 'TimeoutError';
  }
}

export class FrameError extends DocDBError {
  constructor(message: string) {
    super(message, Status.Error);
    this.name = 'FrameError';
  }
}
