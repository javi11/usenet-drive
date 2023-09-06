import {
    createStyles,
    Badge,
    Group,
    Title,
    SimpleGrid,
    Container,
    rem,
} from '@mantine/core';
import { DiskUsage } from '../components/DiskUsageCard';
import { UsenetConnections } from '../components/UsenetConnectionsCard';

const useStyles = createStyles((theme) => ({
    title: {
        fontSize: rem(34),
        fontWeight: 900,

        [theme.fn.smallerThan('sm')]: {
            fontSize: rem(24),
        },
    },

    description: {
        maxWidth: 600,
        margin: 'auto',

        '&::after': {
            content: '""',
            display: 'block',
            backgroundColor: theme.fn.primaryColor(),
            width: rem(45),
            height: rem(2),
            marginTop: theme.spacing.sm,
            marginLeft: 'auto',
            marginRight: 'auto',
        },
    },
}));

export default function Home() {
    const { classes } = useStyles();

    return (
        <Container size="lg" py="xl">
            <Group position="center">
                <Badge variant="filled" size="lg">
                    Server Info
                </Badge>
            </Group>

            <Title order={2} className={classes.title} ta="center" mt="sm">
                Welcome to usenet drive
            </Title>

            <SimpleGrid cols={2} spacing="xl" mt={50} breakpoints={[{ maxWidth: 'md', cols: 1 }]}>
                <DiskUsage />
                <UsenetConnections />
            </SimpleGrid>
        </Container>
    );
}