ALTER TABLE activity_logs
 ADD CONSTRAINT fk_activity_project_team
 FOREIGN KEY (team_id,project_id) REFERENCES projects(team_id,id);
