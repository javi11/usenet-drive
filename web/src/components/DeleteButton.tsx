import {
    Button,
    IconButton,
    AlertDialog,
    AlertDialogBody,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogContent,
    AlertDialogOverlay,
} from "@chakra-ui/react";
import React, { useState } from 'react';
import { MdDelete } from "react-icons/md";

interface DeleteButtonProps {
    itemId: number;
    onDeleteItem: (id: number) => void;
}

export default function DeleteButton({ itemId, onDeleteItem }: DeleteButtonProps) {
    const [isOpen, setIsOpen] = useState(false);
    const onClose = () => setIsOpen(false);
    const cancelRef = React.useRef<HTMLButtonElement>(null);

    return (
        <>
            <IconButton aria-label='Delete upload job' icon={<MdDelete />} colorScheme='pink' variant='solid' onClick={() => setIsOpen(true)} />
            <AlertDialog isOpen={isOpen} leastDestructiveRef={cancelRef} onClose={onClose}>
                <AlertDialogOverlay>
                    <AlertDialogContent>
                        <AlertDialogHeader fontSize="lg" fontWeight="bold">
                            Delete Job
                        </AlertDialogHeader>

                        <AlertDialogBody>
                            Are you sure? You can't undo this action afterwards.
                        </AlertDialogBody>

                        <AlertDialogFooter>
                            <Button ref={cancelRef} onClick={onClose}>
                                Cancel
                            </Button>
                            <Button colorScheme="red" onClick={async () => {
                                await onDeleteItem(itemId)
                                onClose()
                            }} ml={3}>
                                Delete
                            </Button>
                        </AlertDialogFooter>
                    </AlertDialogContent>
                </AlertDialogOverlay>
            </AlertDialog>
        </>
    )
}