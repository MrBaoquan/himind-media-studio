import {StrictMode} from 'react';
import {createRoot} from 'react-dom/client';
import {App} from './app';
import {applyTheme, readTheme} from './theme';
import './styles.css';

applyTheme(readTheme());

const container = document.getElementById('root');
if (!container) throw new Error('missing #root');

createRoot(container).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
