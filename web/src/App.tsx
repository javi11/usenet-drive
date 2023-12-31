import { Group, AppShell, Header, Image, Burger, Navbar, Stack, Menu, MediaQuery, Title } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import {
  IconHome2,
  IconBrandGithub,
  IconFileBroken,
  IconActivity,
} from '@tabler/icons-react';
import Home from './pages/Home'
import NotFound from './pages/NotFound'
import logo from './assets/logo.png'
import NavbarLink from './components/NavBarLink';
import CorruptedNzbs from './pages/CorruptedNzbs';
import Activity from './pages/Activity';

const paths = [
  { href: '/', icon: IconHome2, label: 'Home', elem: <Home /> },
  { href: '/nzbs/corrupted', icon: IconFileBroken, label: 'List of corrupted NZBs', elem: <CorruptedNzbs /> },
  { href: '/activity', icon: IconActivity, label: 'Download/Upload activity', elem: <Activity /> },
];

export default function App() {
  const navigate = useNavigate();
  const location = useLocation();
  const [opened, { toggle }] = useDisclosure(false);

  const links = paths.map((link) => (
    <NavbarLink
      {...link}
      key={link.label}
      active={link.href === location.pathname}
      onClick={() => {
        navigate(link.href)
      }}
    />
  ));

  const mobileLinks = paths.map((link) => (
    <Menu.Item component="a" href={link.href} icon={<link.icon size="1.2rem" stroke={1.5} />}>
      {link.label}
    </Menu.Item>
  ));

  return (
    <AppShell
      padding="md"
      navbarOffsetBreakpoint="sm"
      navbar={
        <MediaQuery smallerThan="sm" styles={{ display: 'none' }}>
          <Navbar p="md" hiddenBreakpoint="sm" width={{ base: 80 }}>
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
        </MediaQuery>
      }
      header={<Header height={90} p="xs">
        <Group position="apart">
          <Group position="apart">
            <Image height={70} width={70} src={logo} alt='Usenet drive' />
            <Title order={2}>Usenet Drive</Title>
          </Group>
          <MediaQuery largerThan="sm" styles={{ display: 'none' }}>
            <Menu shadow="md" onClose={toggle}>
              <Menu.Target>
                <Burger opened={opened} onClick={toggle} size="sm" />
              </Menu.Target>
              <Menu.Dropdown>
                {mobileLinks}
              </Menu.Dropdown>
            </Menu>
          </MediaQuery>
        </Group>
      </Header>}
      styles={(theme) => ({
        main: { backgroundColor: theme.colorScheme === 'dark' ? theme.colors.dark[8] : theme.colors.gray[0] },
      })}
    >
      <Routes>
        {paths.map((link) => (
          <Route key={link.href} path={link.href} element={link.elem} />
        ))}
        <Route key="not-found" path="*" element={<NotFound />} />
      </Routes>
    </AppShell>
  );
}