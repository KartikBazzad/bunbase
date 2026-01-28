import { RequestFrame, ResponseFrame, OperationType } from "../types/protocol";
import {
  SIZES,
  FRAME_OVERHEAD,
  OP_OVERHEAD,
  RESPONSE_OVERHEAD,
  MAX_FRAME_SIZE,
} from "./constants";
import {
  writeLittleEndianUint32,
  writeLittleEndianUint64,
  writeLittleEndianUint16,
  stringToUint8Array,
} from "../utils/buffer";
import { ValidationError, FrameError } from "../types/errors";

export class ProtocolEncoder {
  static encodeRequest(frame: RequestFrame): Uint8Array {
    let size = FRAME_OVERHEAD;

    for (const op of frame.ops) {
      size += SIZES.OP_TYPE;

      // Collection name
      const collection = op.collection || "_default";
      const collectionBytes = stringToUint8Array(collection);
      size += SIZES.COLLECTION_LEN + collectionBytes.length;

      size += SIZES.DOC_ID;

      // Patch operations (for OpPatch)
      if (op.opType === OperationType.Patch && op.patchOps) {
        const patchOpsJSON = JSON.stringify(op.patchOps);
        const patchOpsBytes = stringToUint8Array(patchOpsJSON);
        size += SIZES.PATCH_OPS_LEN + patchOpsBytes.length;
      }

      size += SIZES.PAYLOAD_LEN + (op.payload ? op.payload.length : 0);
    }

    if (size > MAX_FRAME_SIZE) {
      throw new ValidationError("Frame size exceeds maximum");
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

      // Encode collection name
      const collection = op.collection || "_default";
      const collectionBytes = stringToUint8Array(collection);
      const collectionLenBuf = writeLittleEndianUint16(collectionBytes.length);
      buf.set(collectionLenBuf, offset);
      offset += SIZES.COLLECTION_LEN;
      if (collectionBytes.length > 0) {
        buf.set(collectionBytes, offset);
        offset += collectionBytes.length;
      }

      const docIDBuf = writeLittleEndianUint64(op.docID);
      buf.set(docIDBuf, offset);
      offset += SIZES.DOC_ID;

      // Encode patch operations (for OpPatch)
      if (op.opType === OperationType.Patch && op.patchOps) {
        const patchOpsJSON = JSON.stringify(op.patchOps);
        const patchOpsBytes = stringToUint8Array(patchOpsJSON);
        const patchOpsLenBuf = writeLittleEndianUint32(patchOpsBytes.length);
        buf.set(patchOpsLenBuf, offset);
        offset += SIZES.PATCH_OPS_LEN;
        if (patchOpsBytes.length > 0) {
          buf.set(patchOpsBytes, offset);
          offset += patchOpsBytes.length;
        }
      }

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
      throw new ValidationError("Frame size exceeds maximum");
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
