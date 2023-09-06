import { CopyButton, Popover, UnstyledButton, Text } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';

interface LargeTextProps {
    text: string;
}

export default function LargeText({ text }: LargeTextProps) {
    const [opened, { close, open }] = useDisclosure(false);
    return (
        <CopyButton value={text}>
            {({ copied, copy }) => (
                <UnstyledButton color={copied ? 'teal' : 'blue'} onClick={copy} onMouseEnter={open} onMouseLeave={close}>
                    <Popover width={200} position="bottom" withArrow shadow="md" opened={opened}>
                        <Popover.Target>
                            <Text c="dimmed">
                                {copied ? 'Path copied' : text.length > 100 ? text.substring(0, 100) + '...' : text}
                            </Text>
                        </Popover.Target>
                        <Popover.Dropdown>
                            <Text size="sm">{text}</Text>
                        </Popover.Dropdown>
                    </Popover>
                </UnstyledButton>
            )}
        </CopyButton>
    )
}