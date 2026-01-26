import { describe, expect, test } from 'bun:test';
import { ProtocolEncoder } from '../../src/protocol/encoder';
import { OperationType, Command, Status } from '../../src/types/protocol';
import { readLittleEndianUint32, writeLittleEndianUint32, stringToUint8Array, uint8ArrayToString } from '../../src/utils/buffer';
import { JSONUtils } from '../../src/utils/json';

describe('Protocol Encoding', () => {
  test('should encode request frame', () => {
    const frame = {
      requestID: 1n,
      dbID: 0n,
      command: Command.OpenDB,
      opCount: 1,
      ops: [{
        opType: OperationType.Create,
        docID: 0n,
        payload: new TextEncoder().encode('testdb')
      }]
    };

    const encoded = ProtocolEncoder.encodeRequest(frame);
    expect(encoded).toBeInstanceOf(Uint8Array);
    expect(encoded.length).toBeGreaterThan(0);
  });

  test('should encode response frame', () => {
    const frame = {
      requestID: 1n,
      status: Status.OK,
      data: new TextEncoder().encode('response')
    };

    const encoded = ProtocolEncoder.encodeResponse(frame);
    expect(encoded).toBeInstanceOf(Uint8Array);
    expect(encoded.length).toBeGreaterThan(0);
  });
});

describe('Buffer Utilities', () => {
  test('should encode and decode uint32', () => {
    const value = 42;
    const encoded = writeLittleEndianUint32(value);
    const decoded = readLittleEndianUint32(encoded);
    expect(decoded).toBe(value);
  });

  test('should convert string to Uint8Array and back', () => {
    const str = 'Hello, DocDB!';
    const encoded = stringToUint8Array(str);
    const decoded = uint8ArrayToString(encoded);
    expect(decoded).toBe(str);
  });
});

describe('JSON Utilities', () => {
  test('should encode and decode JSON', () => {
    const obj = { foo: 'bar', num: 42 };
    const encoded = JSONUtils.toJSON(obj);
    const decoded = JSONUtils.fromJSON<typeof obj>(encoded);
    expect(decoded).toEqual(obj);
  });

  test('should handle complex JSON objects', () => {
    const obj = {
      id: 1,
      name: 'Test',
      nested: {
        value: 123
      },
      array: [1, 2, 3]
    };
    const encoded = JSONUtils.toJSON(obj);
    const decoded = JSONUtils.fromJSON<typeof obj>(encoded);
    expect(decoded).toEqual(obj);
  });
});
