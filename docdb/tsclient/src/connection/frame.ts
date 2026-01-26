import { SocketConnection } from './socket';
import { writeLittleEndianUint32, readLittleEndianUint32 } from '../utils/buffer';
import { FrameError } from '../types/errors';
import { MAX_FRAME_SIZE } from '../protocol/constants';

export class FrameHandler {
  private conn: SocketConnection;

  constructor(conn: SocketConnection) {
    this.conn = conn;
  }

  async writeFrame(data: Uint8Array): Promise<void> {
    if (data.length > MAX_FRAME_SIZE) {
      throw new FrameError('Frame size exceeds maximum');
    }

    const lenBuf = new Uint8Array(4);
    const view = new DataView(lenBuf.buffer);
    view.setUint32(0, data.length, true);

    await this.conn.write(lenBuf);
    await this.conn.write(data);
  }

  async readFrame(): Promise<Uint8Array> {
    const lenBuf = await this.conn.read(4);
    const length = readLittleEndianUint32(lenBuf);

    if (length > MAX_FRAME_SIZE) {
      throw new FrameError('Frame size exceeds maximum');
    }

    return await this.conn.read(length);
  }
}
