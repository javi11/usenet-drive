import { useState } from 'react';
import { Navbar, Tooltip, UnstyledButton, createStyles, Stack, rem } from '@mantine/core';
import {
    IconHome2,
    IconProgress,
    IconClock,
    IconExclamationCircle,
    IconBrandGithub,
} from '@tabler/icons-react';
import { useNavigate } from 'react-router-dom';

const useStyles = createStyles((theme) => ({
    link: {
        width: rem(50),
        height: rem(50),
        borderRadius: theme.radius.md,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: theme.colorScheme === 'dark' ? theme.colors.dark[0] : theme.colors.gray[7],

        '&:hover': {
            backgroundColor: theme.colorScheme === 'dark' ? theme.colors.dark[5] : theme.colors.gray[0],
        },
    },

    active: {
        '&, &:hover': {
            backgroundColor: theme.fn.variant({ variant: 'light', color: theme.primaryColor }).background,
            color: theme.fn.variant({ variant: 'light', color: theme.primaryColor }).color,
        },
    },
}));

interface NavbarLinkProps {
    icon: React.FC<any>;
    label: string;
    active?: boolean;
    onClick?(): void;
}

function NavbarLink({ icon: Icon, label, active, onClick }: NavbarLinkProps) {
    const { classes, cx } = useStyles();
    return (
        <Tooltip label={label} position="right" transitionProps={{ duration: 0 }}>
            <UnstyledButton onClick={onClick} className={cx(classes.link, { [classes.active]: active })}>
                <Icon size="1.2rem" stroke={1.5} />
            </UnstyledButton>
        </Tooltip>
    );
}

const mockdata = [
    { href: '/', icon: IconHome2, label: 'Home' },
    { href: '/in-progress', icon: IconProgress, label: 'In progress jobs' },
    { href: '/pending', icon: IconClock, label: 'Pending jobs' },
    { href: '/failed', icon: IconExclamationCircle, label: 'Failed jobs' },
];

export default function CustomNavbar() {
    const navigate = useNavigate();
    const [active, setActive] = useState(0);

    const links = mockdata.map((link, index) => (
        <NavbarLink
            {...link}
            key={link.label}
            active={index === active}
            onClick={() => {
                setActive(index);
                navigate(link.href)
            }}
        />
    ));

    return (
        <Navbar width={{ base: 80 }} p="md">
            <Navbar.Section grow mt={50}>
                <Stack justify="center" spacing={0}>
                    {links}
                </Stack>
            </Navbar.Section>
            <Navbar.Section>
                <Stack justify="center" spacing={0}>
                    <NavbarLink icon={IconBrandGithub} label="See it on github" onClick={() => window.open("https://github.com/javi11/usenet-drive", "_blank", "noreferrer")} />
                </Stack>
            </Navbar.Section>
        </Navbar>
    );
}