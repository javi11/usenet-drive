import { Container, Title, rem } from '@mantine/core';

const useStyles = createStyles((theme) => ({
    wrapper: {
        display: 'flex',
        alignItems: 'center',
        padding: `calc(${theme.spacing.xl} * 2)`,
        borderRadius: theme.radius.md,
        backgroundColor: theme.colorScheme === 'dark' ? theme.colors.dark[8] : theme.white,
        border: `${rem(1)} solid ${theme.colorScheme === 'dark' ? theme.colors.dark[8] : theme.colors.gray[3]
            }`,
        flexDirection: 'column',
    },
    title: {
        color: theme.colorScheme === 'dark' ? theme.white : theme.black,
        fontFamily: `Greycliff CF, ${theme.fontFamily}`,
        lineHeight: 1,
        marginBottom: theme.spacing.md,
    },
}));

export default function ManualProcess() {
    const { classes } = useStyles();
    return (
        <Container size="lg" className={classes.wrapper}>
            <Title align="center" className={classes.title}>
                Jobs in progress
            </Title>
        </Container>
    );
}