import {
  RequestFrame,
  ResponseFrame,
  OperationType,
  Status,
  PatchOperation,
} from "../types/protocol";
import {
  SIZES,
  FRAME_OVERHEAD,
  OP_OVERHEAD,
  RESPONSE_OVERHEAD,
  MAX_FRAME_SIZE,
} from "./constants";
import {
  readLittleEndianUint16,
  readLittleEndianUint32,
  readLittleEndianUint64,
  uint8ArrayToString,
} from "../utils/buffer";
import { ValidationError, FrameError } from "../types/errors";

export class ProtocolDecoder {
  static decodeRequest(data: Uint8Array): RequestFrame {
    if (data.length < FRAME_OVERHEAD) {
      throw new ValidationError("Invalid request frame: too short");
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
      if (offset + SIZES.OP_TYPE + SIZES.COLLECTION_LEN > data.length) {
        throw new ValidationError("Invalid operation: incomplete header");
      }

      const op: any = {
        opType: data[offset] as OperationType,
        docID: 0n,
        payload: null as Uint8Array | null,
      };
      offset += SIZES.OP_TYPE;

      // Decode collection name
      const collectionLen = readLittleEndianUint16(data, offset);
      offset += SIZES.COLLECTION_LEN;
      if (offset + collectionLen > data.length) {
        throw new ValidationError(
          "Invalid operation: incomplete collection name",
        );
      }
      if (collectionLen > 0) {
        op.collection = uint8ArrayToString(
          data.subarray(offset, offset + collectionLen),
        );
        offset += collectionLen;
      } else {
        op.collection = "_default";
      }

      if (offset + SIZES.DOC_ID > data.length) {
        throw new ValidationError("Invalid operation: incomplete docID");
      }
      op.docID = readLittleEndianUint64(data, offset);
      offset += SIZES.DOC_ID;

      // Decode patch operations (for OpPatch)
      if (op.opType === OperationType.Patch) {
        if (offset + SIZES.PATCH_OPS_LEN > data.length) {
          throw new ValidationError(
            "Invalid operation: incomplete patch ops length",
          );
        }
        const patchOpsLen = readLittleEndianUint32(data, offset);
        offset += SIZES.PATCH_OPS_LEN;
        if (offset + patchOpsLen > data.length) {
          throw new ValidationError("Invalid operation: incomplete patch ops");
        }
        if (patchOpsLen > 0) {
          const patchOpsJSON = uint8ArrayToString(
            data.subarray(offset, offset + patchOpsLen),
          );
          op.patchOps = JSON.parse(patchOpsJSON) as PatchOperation[];
          offset += patchOpsLen;
        }
      }

      if (offset + SIZES.PAYLOAD_LEN > data.length) {
        throw new ValidationError(
          "Invalid operation: incomplete payload length",
        );
      }
      const payloadLen = readLittleEndianUint32(data, offset);
      offset += SIZES.PAYLOAD_LEN;

      if (offset + payloadLen > data.length) {
        throw new ValidationError("Invalid operation: incomplete payload");
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
      throw new ValidationError("Invalid response frame: too short");
    }

    const frame: ResponseFrame = {
      requestID: readLittleEndianUint64(data, 0),
      status: data[8] as Status,
      data: new Uint8Array(0),
    };

    const dataLen = readLittleEndianUint32(data, 9);

    if (13 + dataLen > data.length) {
      throw new ValidationError("Invalid response: incomplete data");
    }

    if (dataLen > 0) {
      frame.data = data.subarray(13, 13 + dataLen);
    }

    return frame;
  }

  static parseReadResponse(data: Uint8Array): Uint8Array {
    if (data.length < 4) {
      throw new FrameError("Invalid read response: too short");
    }

    const count = readLittleEndianUint32(data, 0);
    if (count !== 1) {
      throw new FrameError("Invalid read response: expected 1 result");
    }

    const payloadLen = readLittleEndianUint32(data, 4);
    if (4 + payloadLen > data.length) {
      throw new FrameError("Invalid read response: incomplete payload");
    }

    return data.subarray(8, 8 + payloadLen);
  }

  static parseBatchResponse(data: Uint8Array): Uint8Array[] {
    if (data.length < 4) {
      throw new FrameError("Invalid batch response: too short");
    }

    const count = readLittleEndianUint32(data, 0);
    const responses: Uint8Array[] = [];
    let offset = 4;

    for (let i = 0; i < count; i++) {
      if (offset + 4 > data.length) {
        throw new FrameError("Invalid batch response: incomplete header");
      }

      const payloadLen = readLittleEndianUint32(data, offset);
      offset += 4;

      if (offset + payloadLen > data.length) {
        throw new FrameError("Invalid batch response: incomplete payload");
      }

      responses.push(data.subarray(offset, offset + payloadLen));
      offset += payloadLen;
    }

    return responses;
  }

  static parseStats(data: Uint8Array) {
    if (data.length !== 40) {
      throw new FrameError("Invalid stats response: expected 40 bytes");
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
