import {
  EuiAvatar,
  EuiBasicTable,
  EuiButton,
  EuiCallOut,
  EuiFieldSearch,
  EuiFlexGroup,
  EuiFlexItem,
  EuiLoadingSpinner,
  EuiPageTemplate,
  EuiPanel,
  EuiSpacer,
  EuiText,
  EuiTitle,
} from '@elastic/eui';

import {
  Outlet,
  RouterProvider,
  createBrowserRouter,
  useLoaderData,
} from "react-router-dom";

import { Fragment, useEffect, useState } from 'react';
import { init as initAPM } from '@elastic/apm-rum';

import { useAuth, AuthProvider } from './auth'

const router = createBrowserRouter([
  {
    path: "/",
    id: "root",
    element: <Root/>,
    loader: async () => {
      return fetch("/api/config").then(response => {
        if (response.ok) {return response.json();}
      })
    },
    children: [
      {
        index: true,
        element: <MainPage/>,
      },
      {
        path: "/auth",
        element: <AuthPage/>,
      },
    ],
  },
]);

export default function App() {
  return <RouterProvider router={router} />;
}

function Root() {
  const config = useLoaderData();
  // Initialize APM for real user monitoring
  initAPM({
    serviceName: 'app-frontend',
    serverUrl: config.apm.server_url,
    propagateTracestate: true,
    breakdownMetrics: true,
    apiVersion: 3,
  });
  return (
    <AuthProvider config={config}>
      <PageLayout><Outlet/></PageLayout>
    </AuthProvider>
  )
}

function PageLayout() {
  const {profile, signInButtonRef} = useAuth();
  
  if (!profile) {
    return (
      <EuiPageTemplate panelled={true}>
        <EuiPageTemplate.Header
             pageTitle="App Scaffold"
             iconType="logoElastic"
             rightSideItems={[<div key="signin" ref={signInButtonRef}></div>]}/>
        <EuiPageTemplate.Section>
          <EuiCallOut title="Welcome" color="primary" iconType="user">
            <p>Please sign in with your Google account to continue.</p>
          </EuiCallOut>
        </EuiPageTemplate.Section>
      </EuiPageTemplate>
    )
  }

  const avatar = <EuiAvatar name={profile.name} imageUrl={profile.picture}/>;
  return (
    <EuiPageTemplate panelled={true}>
      <EuiPageTemplate.Header
           pageTitle="App Scaffold"
           iconType="logoElastic"
           rightSideItems={[avatar]}/>
      <EuiPageTemplate.Section restrictWidth={1400}><Outlet/></EuiPageTemplate.Section>
    </EuiPageTemplate>
  )
}

function MainPage() {
  const {profile} = useAuth();
  
  if (!profile) {
    return null;
  }

  return (
    <Fragment>
      <HelloSection />
      <EuiSpacer size="xl" />
      <DataTableSection />
    </Fragment>
  )
}

function HelloSection() {
  const [helloData, setHelloData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetch("/api/hello")
      .then(response => {
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
      })
      .then(data => {
        setHelloData(data);
        setLoading(false);
      })
      .catch(err => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return (
      <EuiPanel>
        <EuiFlexGroup justifyContent="center" alignItems="center">
          <EuiFlexItem grow={false}>
            <EuiLoadingSpinner size="xl" />
          </EuiFlexItem>
        </EuiFlexGroup>
      </EuiPanel>
    );
  }

  if (error) {
    return (
      <EuiCallOut title="Error loading message" color="danger" iconType="alert">
        <p>{error}</p>
      </EuiCallOut>
    );
  }

  return (
    <EuiPanel paddingSize="l">
      <EuiTitle size="m">
        <h2>Welcome!</h2>
      </EuiTitle>
      <EuiSpacer size="m" />
      <EuiText>
        <p><strong>{helloData.message}</strong></p>
        <p>Logged in as: {helloData.user}</p>
        <p>Server time: {new Date(helloData.timestamp).toLocaleString()}</p>
      </EuiText>
    </EuiPanel>
  );
}

function DataTableSection() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [searchValue, setSearchValue] = useState('');
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [sortField, setSortField] = useState('created_at');
  const [sortDirection, setSortDirection] = useState('desc');

  useEffect(() => {
    fetch("/api/data")
      .then(response => {
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
      })
      .then(data => {
        setData(data);
        setLoading(false);
      })
      .catch(err => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  const refreshData = () => {
    setLoading(true);
    fetch("/api/data")
      .then(response => response.json())
      .then(data => {
        setData(data);
        setLoading(false);
      })
      .catch(err => {
        setError(err.message);
        setLoading(false);
      });
  };

  // Filter data based on search
  const filteredData = data.filter(item => {
    if (!searchValue) return true;
    const searchLower = searchValue.toLowerCase();
    return (
      item.id.toLowerCase().includes(searchLower) ||
      item.name.toLowerCase().includes(searchLower) ||
      item.description.toLowerCase().includes(searchLower) ||
      item.status.toLowerCase().includes(searchLower) ||
      item.category.toLowerCase().includes(searchLower)
    );
  });

  // Sort data
  const sortedData = [...filteredData].sort((a, b) => {
    let aValue = a[sortField];
    let bValue = b[sortField];
    
    if (sortField === 'created_at') {
      aValue = new Date(aValue);
      bValue = new Date(bValue);
    }
    
    if (aValue < bValue) return sortDirection === 'asc' ? -1 : 1;
    if (aValue > bValue) return sortDirection === 'asc' ? 1 : -1;
    return 0;
  });

  // Paginate data
  const startIndex = pageIndex * pageSize;
  const pageData = sortedData.slice(startIndex, startIndex + pageSize);

  const columns = [
    {
      field: 'id',
      name: 'ID',
      sortable: true,
      width: '12%',
    },
    {
      field: 'name',
      name: 'Name',
      sortable: true,
      truncateText: true,
      width: '25%',
    },
    {
      field: 'description',
      name: 'Description',
      sortable: false,
      truncateText: true,
      width: '25%',
    },
    {
      field: 'category',
      name: 'Category',
      sortable: true,
      width: '12%',
    },
    {
      field: 'status',
      name: 'Status',
      sortable: true,
      width: '10%',
      render: (status) => {
        const color = {
          'Active': 'success',
          'Pending': 'warning',
          'Completed': 'primary',
          'On Hold': 'default',
          'Cancelled': 'danger',
        }[status] || 'default';
        return <span style={{color: color === 'success' ? 'green' : color === 'danger' ? 'red' : color === 'warning' ? 'orange' : 'inherit'}}>{status}</span>;
      }
    },
    {
      field: 'created_at',
      name: 'Created',
      sortable: true,
      width: '16%',
      render: (date) => new Date(date).toLocaleDateString(),
    },
  ];

  const pagination = {
    pageIndex,
    pageSize,
    totalItemCount: filteredData.length,
    pageSizeOptions: [10, 25, 50],
  };

  const sorting = {
    sort: {
      field: sortField,
      direction: sortDirection,
    },
  };

  const onTableChange = ({ page, sort }) => {
    if (page) {
      setPageIndex(page.index);
      setPageSize(page.size);
    }
    if (sort) {
      setSortField(sort.field);
      setSortDirection(sort.direction);
    }
  };

  if (error) {
    return (
      <EuiCallOut title="Error loading data" color="danger" iconType="alert">
        <p>{error}</p>
      </EuiCallOut>
    );
  }

  return (
    <Fragment>
      <EuiFlexGroup justifyContent="spaceBetween" alignItems="center">
        <EuiFlexItem>
          <EuiTitle size="m">
            <h2>Sample Data</h2>
          </EuiTitle>
        </EuiFlexItem>
        <EuiFlexItem grow={false}>
          <EuiButton onClick={refreshData} iconType="refresh" isLoading={loading}>
            Refresh
          </EuiButton>
        </EuiFlexItem>
      </EuiFlexGroup>
      <EuiSpacer size="m" />
      <EuiFieldSearch
        placeholder="Search..."
        value={searchValue}
        onChange={(e) => {
          setSearchValue(e.target.value);
          setPageIndex(0);
        }}
        isClearable
        fullWidth
      />
      <EuiSpacer size="m" />
      <EuiBasicTable
        tableCaption="Sample data table"
        items={pageData}
        columns={columns}
        pagination={pagination}
        sorting={sorting}
        onChange={onTableChange}
        loading={loading}
      />
    </Fragment>
  );
}

function AuthPage() {
  const {profile, authorizeGoogle, googleAuthorized, googleAuthorizationError} = useAuth();
  
  if (!profile) {
    return (
      <EuiCallOut title="Please sign in" color="primary" iconType="user">
        <p>You need to sign in to access this page.</p>
      </EuiCallOut>
    );
  }

  if (googleAuthorized) {
    return (
      <EuiCallOut title="Authorized" color="success" iconType="check">
        <p>You are fully authorized. You can now use all features of the application.</p>
      </EuiCallOut>
    );
  }

  return (
    <Fragment>
      <EuiCallOut title="Additional Authorization Required" color="warning" iconType="help">
        <p>Some features require additional Google authorization.</p>
        {googleAuthorizationError && <p>Error: {googleAuthorizationError}</p>}
      </EuiCallOut>
      <EuiSpacer />
      <EuiButton onClick={authorizeGoogle}>
        Authorize with Google
      </EuiButton>
    </Fragment>
  );
}
