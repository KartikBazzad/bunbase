import { RequestFrame, ResponseFrame, OperationType } from '../types/protocol';
import { SIZES, FRAME_OVERHEAD, OP_OVERHEAD, RESPONSE_OVERHEAD, MAX_FRAME_SIZE } from './constants';
import { writeLittleEndianUint32, writeLittleEndianUint64 } from '../utils/buffer';
import { ValidationError, FrameError } from '../types/errors';

export class ProtocolEncoder {
  static encodeRequest(frame: RequestFrame): Uint8Array {
    let size = FRAME_OVERHEAD;

    for (const op of frame.ops) {
      size += OP_OVERHEAD + (op.payload ? op.payload.length : 0);
    }

    if (size > MAX_FRAME_SIZE) {
      throw new ValidationError('Frame size exceeds maximum');
    }

    const buf = new Uint8Array(size);
    let offset = 0;

    const reqIDBuf = writeLittleEndianUint64(frame.requestID);
    buf.set(reqIDBuf, offset);
    offset += SIZES.REQUEST_ID;

    const dbIDBuf = writeLittleEndianUint64(frame.dbID);
    buf.set(dbIDBuf, offset);
    offset += SIZES.DB_ID;

    buf[offset] = frame.command;
    offset += SIZES.COMMAND;

    const opCountBuf = writeLittleEndianUint32(frame.opCount);
    buf.set(opCountBuf, offset);
    offset += SIZES.OP_COUNT;

    for (const op of frame.ops) {
      buf[offset] = op.opType;
      offset += SIZES.OP_TYPE;

      const docIDBuf = writeLittleEndianUint64(op.docID);
      buf.set(docIDBuf, offset);
      offset += SIZES.DOC_ID;

      const payloadLen = op.payload ? op.payload.length : 0;
      const payloadLenBuf = writeLittleEndianUint32(payloadLen);
      buf.set(payloadLenBuf, offset);
      offset += SIZES.PAYLOAD_LEN;

      if (op.payload && payloadLen > 0) {
        buf.set(op.payload, offset);
        offset += payloadLen;
      }
    }

    return buf;
  }

  static encodeResponse(frame: ResponseFrame): Uint8Array {
    const size = RESPONSE_OVERHEAD + frame.data.length;

    if (size > MAX_FRAME_SIZE) {
      throw new ValidationError('Frame size exceeds maximum');
    }

    const buf = new Uint8Array(size);
    let offset = 0;

    const reqIDBuf = writeLittleEndianUint64(frame.requestID);
    buf.set(reqIDBuf, offset);
    offset += SIZES.REQUEST_ID;

    buf[offset] = frame.status;
    offset += 1;

    const dataLenBuf = writeLittleEndianUint32(frame.data.length);
    buf.set(dataLenBuf, offset);
    offset += 4;

    if (frame.data.length > 0) {
      buf.set(frame.data, offset);
    }

    return buf;
  }
}
