import React from 'react';
import {
  Page,
  Masthead,
  MastheadMain,
  MastheadBrand,
  MastheadContent,
  PageSidebar,
  PageSidebarBody,
  Nav,
  NavList,
  NavItem,
} from '@patternfly/react-core';
import { useLocation, useNavigate } from 'react-router-dom';

interface AppLayoutProps {
  children: React.ReactNode;
}

export const AppLayout: React.FC<AppLayoutProps> = ({ children }) => {
  const location = useLocation();
  const navigate = useNavigate();

  const Header = (
    <Masthead>
      <MastheadMain>
        <MastheadBrand onClick={() => navigate('/')} style={{ cursor: 'pointer', display: 'flex', alignItems: 'center' }}>
          <img 
            src="/logo.png" 
            alt="Argo Ephemeral" 
            style={{ height: '65px' }}
          />
          <span style={{ marginLeft: '12px', fontSize: '1.2rem', fontWeight: 600, color: '#ffffff' }}>
            Argo Ephemeral Operator
          </span>
        </MastheadBrand>
      </MastheadMain>
      <MastheadContent>{/* User menu, notifications */}</MastheadContent>
    </Masthead>
  );

  const Sidebar = (
    <PageSidebar>
      <PageSidebarBody>
        <Nav>
          <NavList>
            <NavItem
              isActive={location.pathname === '/environments'}
              onClick={() => navigate('/environments')}
            >
              Environments
            </NavItem>
            <NavItem
              isActive={location.pathname === '/settings'}
              onClick={() => navigate('/settings')}
            >
              Settings
            </NavItem>
          </NavList>
        </Nav>
      </PageSidebarBody>
    </PageSidebar>
  );

  return (
    <Page header={Header} sidebar={Sidebar} isManagedSidebar>
      {children}
    </Page>
  );
};
