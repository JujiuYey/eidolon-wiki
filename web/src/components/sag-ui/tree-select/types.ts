export interface TreeSelectNodeLike {
  id: string;
  children?: any[];
}

export interface TreeSelectItem<T extends TreeSelectNodeLike = TreeSelectNodeLike> {
  value: string;
  label: string;
  node: T;
  children?: TreeSelectItem<T>[];
}
