import { Group, AppShell, Header, Image } from '@mantine/core';
import { Routes, Route } from 'react-router-dom';
import Home from './pages/Home'
import FailedJobs from './pages/FailedJobs'
import PendingJobs from './pages/PendingJobs'
import InProgressJobs from './pages/InProgressJobs'
import NotFound from './pages/NotFound'
import reactLogo from './assets/logo.svg'
import CustomNavbar from './components/NavBar';


export default function App() {


  return (
    <AppShell
      padding="md"
      navbar={
        <CustomNavbar />
      }
      header={<Header height={90} p="xs">
        <Group position="apart">
          <Image height={70} width={120} src={reactLogo} alt='Usenet drive' />
        </Group>
      </Header>}
      styles={(theme) => ({
        main: { backgroundColor: theme.colorScheme === 'dark' ? theme.colors.dark[8] : theme.colors.gray[0] },
      })}
    >
      <Routes>
        <Route index path="/" element={<Home />} />
        <Route path="/in-progress" element={<InProgressJobs />} />
        <Route path="/pending" element={<PendingJobs />} />
        <Route path="/failed" element={<FailedJobs />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </AppShell>
  );
}