create table if not exists `executions`
(
    `id`          bigint       not null auto_increment primary key comment 'id',
    `task_name`   varchar(128) not null,
    `task_owners` varchar(512) not null,
    `status`      varchar(32)  not null,
    `retry_count` int          not null default 0,
    `payload`     text,
    `result`      text,
    `created_at`  timestamp    not null default current_timestamp,
    `updated_at`  timestamp    not null default current_timestamp on update current_timestamp,
    `deleted_at`  timestamp    null
) engine InnoDB
  character set utf8mb4
  collate utf8mb4_general_ci;