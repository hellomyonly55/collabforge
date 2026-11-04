export interface Comment {
  id: string;
  task_id: string;
  author_id: string;
  author_nickname: string;
  content: string;
  version: number;
  can_edit: boolean;
  can_delete: boolean;
  created_at: string;
  updated_at: string;
}
