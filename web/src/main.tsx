import React from 'react';
import ReactDOM from 'react-dom/client';
import { MantineProvider } from '@mantine/core';
import { BrowserRouter } from 'react-router-dom';
import { Notifications } from '@mantine/notifications';
import App from './App.tsx';
import { ModalsProvider } from '@mantine/modals';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <MantineProvider>
      <ModalsProvider>
        <BrowserRouter>
          <App />
          <Notifications />
        </BrowserRouter>
      </ ModalsProvider>
    </MantineProvider>
  </React.StrictMode>
);
