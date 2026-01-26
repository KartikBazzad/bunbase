/**
 * FieldValue - Special field value helpers
 */

export class FieldValue {
  private constructor(
    public readonly _type: string,
    public readonly _value?: any,
  ) {}

  /**
   * Returns a sentinel for use with update() to set a field to the server timestamp.
   */
  static serverTimestamp(): FieldValue {
    return new FieldValue("serverTimestamp");
  }

  /**
   * Returns a sentinel for use with update() to increment a numeric field value.
   */
  static increment(n: number): FieldValue {
    return new FieldValue("increment", n);
  }

  /**
   * Returns a sentinel for use with update() to add elements to an array field.
   */
  static arrayUnion(...elements: any[]): FieldValue {
    return new FieldValue("arrayUnion", elements);
  }

  /**
   * Returns a sentinel for use with update() to remove elements from an array field.
   */
  static arrayRemove(...elements: any[]): FieldValue {
    return new FieldValue("arrayRemove", elements);
  }

  /**
   * Returns a sentinel for use with update() to delete a field.
   */
  static delete(): FieldValue {
    return new FieldValue("delete");
  }
}
