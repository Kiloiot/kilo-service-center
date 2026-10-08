/** A page of a server-paginated list. */
export interface ListPage<T> {
  items: T[];
  totalCount: number;
  nextPageToken?: string;
}

/** A 0-based page of an offset-paginated list. */
export interface PageRequest {
  page: number;
  pageSize: number;
}
