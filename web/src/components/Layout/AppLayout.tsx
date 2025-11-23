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
  Brand,
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
        <MastheadBrand onClick={() => navigate('/')} style={{ cursor: 'pointer' }}>
          <Brand
            src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Ctext y='20' font-size='20'%3E🚀%3C/text%3E%3C/svg%3E"
            alt="Argo Ephemeral"
            heights={{ default: '36px' }}
          />
          <span style={{ marginLeft: '12px', fontSize: '1.2rem', fontWeight: 600 }}>
            Argo Ephemeral
          </span>
        </MastheadBrand>
      </MastheadMain>
      <MastheadContent>{/* Can add user menu, notifications, etc */}</MastheadContent>
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

