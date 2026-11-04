import { Component, ReactNode } from 'react';

// Without this, React unmounts the whole app on a render error and leaves a white page.
export default class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  componentDidCatch(error: unknown) {
    console.error(error);
  }

  render() {
    if (!this.state.failed) return this.props.children;
    return (
      <div role="alert" style={{ minHeight: '100dvh', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 16, padding: 24, textAlign: 'center' }}>
        <h1 className="display" style={{ margin: 0, fontSize: 28, fontWeight: 800 }}>Esta página não abriu</h1>
        <p style={{ margin: 0, maxWidth: 360, color: '#57524B', lineHeight: 1.5 }}>Os teus dados estão seguros. Recarrega a página e, se continuar, volta ao início.</p>
        <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', justifyContent: 'center' }}>
          <button className="btn btn-primary" onClick={() => location.reload()}>Recarregar</button>
          <a className="btn" href="/" style={{ display: 'inline-flex', alignItems: 'center' }}>Ir para o início</a>
        </div>
      </div>
    );
  }
}
