export function readLittleEndianUint32(buf: Uint8Array, offset: number = 0): number {
  return buf[offset] | (buf[offset + 1] << 8) | (buf[offset + 2] << 16) | (buf[offset + 3] << 24);
}

export function readLittleEndianUint64(buf: Uint8Array, offset: number = 0): bigint {
  const low = buf[offset] | (buf[offset + 1] << 8) | (buf[offset + 2] << 16) | (buf[offset + 3] << 24);
  const high = buf[offset + 4] | (buf[offset + 5] << 8) | (buf[offset + 6] << 16) | (buf[offset + 7] << 24);
  return BigInt(high) * 4294967296n + BigInt(low);
}

export function writeLittleEndianUint32(value: number): Uint8Array {
  const buf = new Uint8Array(4);
  buf[0] = value & 0xff;
  buf[1] = (value >> 8) & 0xff;
  buf[2] = (value >> 16) & 0xff;
  buf[3] = (value >> 24) & 0xff;
  return buf;
}

export function writeLittleEndianUint64(value: bigint): Uint8Array {
  const buf = new Uint8Array(8);
  const low = Number(value & 0xffffffffn);
  const high = Number((value >> 32n) & 0xffffffffn);

  buf[0] = low & 0xff;
  buf[1] = (low >> 8) & 0xff;
  buf[2] = (low >> 16) & 0xff;
  buf[3] = (low >> 24) & 0xff;
  buf[4] = high & 0xff;
  buf[5] = (high >> 8) & 0xff;
  buf[6] = (high >> 16) & 0xff;
  buf[7] = (high >> 24) & 0xff;
  return buf;
}

export function stringToUint8Array(str: string): Uint8Array {
  return new TextEncoder().encode(str);
}

export function uint8ArrayToString(arr: Uint8Array): string {
  return new TextDecoder().decode(arr);
}
