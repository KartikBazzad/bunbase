import { RequestFrame, ResponseFrame, OperationType, Status } from '../types/protocol';
import { SIZES, FRAME_OVERHEAD, OP_OVERHEAD, RESPONSE_OVERHEAD, MAX_FRAME_SIZE } from './constants';
import { readLittleEndianUint32, readLittleEndianUint64 } from '../utils/buffer';
import { ValidationError, FrameError } from '../types/errors';

export class ProtocolDecoder {
  static decodeRequest(data: Uint8Array): RequestFrame {
    if (data.length < FRAME_OVERHEAD) {
      throw new ValidationError('Invalid request frame: too short');
    }

    let offset = 0;
    const frame: RequestFrame = {
      requestID: readLittleEndianUint64(data, offset),
      dbID: 0n,
      command: data[offset + 16],
      opCount: readLittleEndianUint32(data, offset + 17),
      ops: [],
    };

    offset = FRAME_OVERHEAD;

    for (let i = 0; i < frame.opCount; i++) {
      if (offset + OP_OVERHEAD > data.length) {
        throw new ValidationError('Invalid operation: incomplete header');
      }

      const op = {
        opType: data[offset] as OperationType,
        docID: readLittleEndianUint64(data, offset + 1),
        payload: null as Uint8Array | null,
      };

      offset += OP_OVERHEAD;

      const payloadLen = readLittleEndianUint32(data, offset - 4);

      if (offset + payloadLen > data.length) {
        throw new ValidationError('Invalid operation: incomplete payload');
      }

      if (payloadLen > 0) {
        op.payload = data.subarray(offset, offset + payloadLen);
        offset += payloadLen;
      }

      frame.ops.push(op);
    }

    return frame;
  }

  static decodeResponse(data: Uint8Array): ResponseFrame {
    if (data.length < RESPONSE_OVERHEAD) {
      throw new ValidationError('Invalid response frame: too short');
    }

    const frame: ResponseFrame = {
      requestID: readLittleEndianUint64(data, 0),
      status: data[8] as Status,
      data: new Uint8Array(0),
    };

    const dataLen = readLittleEndianUint32(data, 9);

    if (13 + dataLen > data.length) {
      throw new ValidationError('Invalid response: incomplete data');
    }

    if (dataLen > 0) {
      frame.data = data.subarray(13, 13 + dataLen);
    }

    return frame;
  }

  static parseReadResponse(data: Uint8Array): Uint8Array {
    if (data.length < 4) {
      throw new FrameError('Invalid read response: too short');
    }

    const count = readLittleEndianUint32(data, 0);
    if (count !== 1) {
      throw new FrameError('Invalid read response: expected 1 result');
    }

    const payloadLen = readLittleEndianUint32(data, 4);
    if (4 + payloadLen > data.length) {
      throw new FrameError('Invalid read response: incomplete payload');
    }

    return data.subarray(8, 8 + payloadLen);
  }

  static parseBatchResponse(data: Uint8Array): Uint8Array[] {
    if (data.length < 4) {
      throw new FrameError('Invalid batch response: too short');
    }

    const count = readLittleEndianUint32(data, 0);
    const responses: Uint8Array[] = [];
    let offset = 4;

    for (let i = 0; i < count; i++) {
      if (offset + 4 > data.length) {
        throw new FrameError('Invalid batch response: incomplete header');
      }

      const payloadLen = readLittleEndianUint32(data, offset);
      offset += 4;

      if (offset + payloadLen > data.length) {
        throw new FrameError('Invalid batch response: incomplete payload');
      }

      responses.push(data.subarray(offset, offset + payloadLen));
      offset += payloadLen;
    }

    return responses;
  }

  static parseStats(data: Uint8Array) {
    if (data.length !== 40) {
      throw new FrameError('Invalid stats response: expected 40 bytes');
    }

    return {
      totalDBs: Number(readLittleEndianUint64(data, 0)),
      activeDBs: Number(readLittleEndianUint64(data, 8)),
      totalTxns: readLittleEndianUint64(data, 16),
      walSize: readLittleEndianUint64(data, 24),
      memoryUsed: readLittleEndianUint64(data, 32),
      memoryCapacity: 0n,
    };
  }
}
