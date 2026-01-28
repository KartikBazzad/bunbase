export class JSONUtils {
  static toJSON(data: unknown): Uint8Array {
    const json = JSON.stringify(data);
    return new TextEncoder().encode(json);
  }

  static fromJSON<T>(data: Uint8Array): T {
    const json = new TextDecoder().decode(data);
    return JSON.parse(json) as T;
  }

  static encodeBytes(buffer: Uint8Array): object {
    const base64 = btoa(String.fromCharCode(...buffer));
    return {
      _type: 'bytes',
      encoding: 'base64',
      data: base64,
    };
  }

  static decodeBytes(obj: any): Uint8Array {
    if (obj._type !== 'bytes') {
      throw new Error('Not a bytes wrapper');
    }
    if (obj.encoding !== 'base64') {
      throw new Error(`Unsupported encoding: ${obj.encoding}`);
    }
    const binaryString = atob(obj.data);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }
    return bytes;
  }
}
