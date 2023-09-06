import { Badge, Progress, Tooltip, rem } from '@mantine/core';
import { IconClock, IconX } from '@tabler/icons-react';
import { JobStatus } from '../data/job';

type StatusProps = {
    status: JobStatus;
    error?: string;
};

const Status = ({ status, error }: StatusProps) => {
    switch (status) {
        case JobStatus.InProgress:
            return <Tooltip label="Uploading">
                <Progress value={100} striped animate />
            </Tooltip>
        case JobStatus.Pending:
            return <Badge pl={0} size="lg" color="grey" radius="xl" leftSection={<IconClock size={rem(10)} />}>
                Pending
            </Badge>;
        case JobStatus.Failed:
            return <Tooltip label={error}>
                <Badge pr={20} color="red" leftSection={<IconX size={rem(10)} />}>
                    Error
                </Badge>
            </Tooltip>;
        default:
            return null;
    }
};

export default Status;
