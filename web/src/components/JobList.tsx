'use client'
import { useState, useCallback } from 'react';
import {
    Flex,
    Table,
    Tbody,
    Text,
    Th,
    Thead,
    Tr,
    useColorModeValue,
    Td,
    Card,
    CardHeader,
    CardBody,
    IconButton,
    useToast,
    Button,
    HStack,
    Tfoot,
    Box,
} from "@chakra-ui/react";
import { CgRedo } from "react-icons/cg";
import Status from './Status';
import { JobResponse } from '../data/job';
import DeleteButton from './DeleteButton';

interface ListProps {
    title: string;
    captions: string[];
    data: JobResponse;
    onPageChange: (page: number) => void;
}

const JobList = ({ title, captions, data, onPageChange }: ListProps) => {
    const totalPages = Math.ceil(data.totalCount / data.limit);
    const currentPage = Math.ceil(data.offset / data.limit) + 1;
    const textColor = useColorModeValue("gray.700", "white");
    const toast = useToast();

    const handleDelete = useCallback(async (id: number) => {
        try {
            const res = await fetch(`/api/jobs/${id}`, { method: "DELETE" });
            if (!res.ok) {
                throw new Error(`Error deleting job ${id}.`);
            }
        } catch (error) {
            toast({
                title: "An error occurred.",
                description: `Unable to delete job ${id}.`,
                status: "error",
                duration: 5000,
                isClosable: true,
            });
        }
    }, [toast]);

    const handleRetry = useCallback(async (id: number) => {
        try {
            const res = await fetch(`/api/jobs/${id}/retry`, { method: "PUT" });
            if (!res.ok) {
                throw new Error(`Error retrying job ${id}.`);
            }
        } catch (error) {
            toast({
                title: "An error occurred.",
                description: `Unable to retry job ${id}.`,
                status: "error",
                duration: 5000,
                isClosable: true,
            });
        }
    }, [toast]);

    return (
        <Card my='22px' overflowX={{ sm: "scroll", xl: "hidden" }}>
            <CardHeader>
                <Flex direction='column'>
                    <Text fontSize='lg' color={textColor} fontWeight='bold' pb='.5rem'>
                        {title}
                    </Text>
                </Flex>
            </CardHeader>
            <CardBody>
                <Table variant='simple' color={textColor}>
                    <Thead>
                        <Tr my='.8rem' pl='0px'>
                            {captions.map((caption, idx) => {
                                return (
                                    <Th color='gray.400' key={idx} ps={idx === 0 ? "0px" : undefined}>
                                        {caption}
                                    </Th>
                                );
                            })}
                        </Tr>
                    </Thead>
                    <Tbody>
                        {data.entries.map((row) => {
                            return (
                                <Tr key={row.id}>
                                    <Td>{row.id}</Td>
                                    <Td><Text fontSize='sm'>{row.data}</Text></Td>
                                    <Td><Text fontSize='sm'>{row.createdAt}</Text></Td>
                                    <Td><Status status={row.status} error={row.error} /></Td>
                                    {captions.includes("actions") ? <Td>
                                        <HStack spacing={4}>
                                            <DeleteButton itemId={row.id} onDeleteItem={handleDelete} />
                                            {captions.includes("error") ? <IconButton aria-label='Retry upload job' icon={<CgRedo />} colorScheme='blue' variant='solid' onClick={() => handleRetry(row.id)} /> : null}
                                        </HStack>
                                    </Td> : null}
                                </Tr>
                            );
                        })}
                    </Tbody>
                    <Tfoot>
                        <Tr>
                            <Td>
                                <HStack mt={5}>
                                    {Array.from({ length: totalPages }, (_, i) => i + 1).map((page) => (
                                        <Button key={page} onClick={() => onPageChange(page)} colorScheme={currentPage === page ? 'blue' : 'gray'}>
                                            {page}
                                        </Button>
                                    ))}
                                </HStack>
                                <Box mt={2}>
                                    Showing {Math.max((currentPage - 1) * data.limit, 1)} to {data.offset + data.limit} of {data.totalCount} jobs
                                </Box>
                            </Td>
                        </Tr>
                    </Tfoot>
                </Table>
            </CardBody>
        </Card>
    );
};

export default JobList;