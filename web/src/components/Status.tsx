import { Spinner, Icon, Text, Tooltip } from "@chakra-ui/react";
import { MdError, MdAccessTime } from "react-icons/md";
import { JobStatus } from '../data/job';

type StatusProps = {
    status: JobStatus;
    error?: string;
};

const Status = ({ status, error }: StatusProps) => {
    switch (status) {
        case JobStatus.InProgress:
            return <Text color="gray.500">Uploading <Spinner size="sm" /> </Text>;
        case JobStatus.Pending:
            return <Icon as={MdAccessTime} />;
        case JobStatus.Failed:
            return <Tooltip hasArrow label={error} aria-label='Error' bg='red.600'>
                <span>
                    <Icon as={MdError} color="red.500" />
                </span>
            </Tooltip>;
        default:
            return null;
    }
};

export default Status;
