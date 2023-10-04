import { Container, LoadingOverlay } from '@mantine/core';
import { useEffect, useState } from 'react';
import { Prism } from '@mantine/prism';

interface FileContentProps {
    contentStream: ReadableStream<Uint8Array>;
}

export default function FileContent({ contentStream }: FileContentProps) {
    const [content, setContent] = useState<string>('');

    useEffect(() => {
        const reader = contentStream.getReader();
        let result = '';

        const readStream = async () => {
            const { done, value } = await reader.read();
            if (done) {
                setContent(result);
                return;
            }
            const decoder = new TextDecoder();
            result += decoder.decode(value);
            readStream();
        };

        readStream();

        return () => {
            reader.cancel();
        };
    }, [contentStream]);

    return (
        <Container>
            {content ? (
                <Prism language="markup" style={{ marginTop: 10 }}>
                    {content}
                </Prism>
            ) : (
                <LoadingOverlay visible />
            )}
        </Container>
    );
}