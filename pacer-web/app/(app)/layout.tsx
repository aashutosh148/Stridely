'use client';

import { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import Link from 'next/link';
import { useUser } from '../../hooks/useUser';
import { ConnectBanner } from '../../components/connect-banner';
import { NotificationToaster } from '../../components/notification-toaster';
import {
  Home,
  Calendar,
  MessageSquare,
  Trophy,
  BarChart2,
  Settings,
  Menu,
  X,
  LogOut
} from 'lucide-react';

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: Home },
  { name: 'Plan', href: '/plan', icon: Calendar },
  { name: 'Chat', href: '/chat', icon: MessageSquare },
  { name: 'Race', href: '/race', icon: Trophy },
  { name: 'Stats', href: '/stats', icon: BarChart2 },
  { name: 'Settings', href: '/settings', icon: Settings },
];

export default function AppLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const router = useRouter();
  const pathname = usePathname();
  const { user, isLoading } = useUser();

  useEffect(() => {
    const token = localStorage.getItem('pacer_token');
    if (!token) {
      router.push('/login');
    }
  }, [router]);

  const handleLogout = () => {
    localStorage.removeItem('pacer_token');
    router.push('/login');
  };

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-indigo-600 border-t-transparent"></div>
      </div>
    );
  }

  // Very basic countdown for target race. Should eventually be fetched from the backend or user object if available.
  const daysToRace = user?.targetRaceId ? 42 : null; 

  return (
    <div>
      {/* Mobile sidebar */}
      {sidebarOpen && (
        <div className="fixed inset-0 z-50 flex lg:hidden">
          <div className="fixed inset-0 bg-gray-900/80" onClick={() => setSidebarOpen(false)} />
          <div className="relative mr-16 flex w-full max-w-xs flex-1 flex-col bg-white">
            <div className="absolute left-full top-0 flex w-16 justify-center pt-5">
              <button type="button" className="-m-2.5 p-2.5" onClick={() => setSidebarOpen(false)}>
                <span className="sr-only">Close sidebar</span>
                <X className="h-6 w-6 text-white" aria-hidden="true" />
              </button>
            </div>
            <div className="flex grow flex-col gap-y-5 overflow-y-auto px-6 pb-4">
              <div className="flex h-16 shrink-0 items-center font-bold text-xl text-indigo-600">
                Pacer
              </div>
              <nav className="flex flex-1 flex-col">
                <ul role="list" className="flex flex-1 flex-col gap-y-7">
                  <li>
                    <ul role="list" className="-mx-2 space-y-1">
                      {navigation.map((item) => {
                        const isCurrent = pathname === item.href;
                        return (
                          <li key={item.name}>
                            <Link
                              href={item.href}
                              className={`
                                group flex gap-x-3 rounded-md p-2 text-sm leading-6 font-semibold
                                ${isCurrent ? 'bg-gray-50 text-indigo-600' : 'text-gray-700 hover:bg-gray-50 hover:text-indigo-600'}
                              `}
                              onClick={() => setSidebarOpen(false)}
                            >
                              <item.icon
                                className={`h-6 w-6 shrink-0 ${isCurrent ? 'text-indigo-600' : 'text-gray-400 group-hover:text-indigo-600'}`}
                                aria-hidden="true"
                              />
                              {item.name}
                            </Link>
                          </li>
                        );
                      })}
                    </ul>
                  </li>
                  <li className="mt-auto">
                    <button
                      onClick={handleLogout}
                      className="group -mx-2 flex w-full gap-x-3 rounded-md p-2 text-sm font-semibold leading-6 text-gray-700 hover:bg-gray-50 hover:text-indigo-600"
                    >
                      <LogOut className="h-6 w-6 shrink-0 text-gray-400 group-hover:text-indigo-600" aria-hidden="true" />
                      Logout
                    </button>
                  </li>
                </ul>
              </nav>
            </div>
          </div>
        </div>
      )}

      {/* Static sidebar for desktop */}
      <div className="hidden lg:fixed lg:inset-y-0 lg:z-50 lg:flex lg:w-72 lg:flex-col">
        <div className="flex grow flex-col gap-y-5 overflow-y-auto border-r border-gray-200 bg-white px-6 pb-4">
          <div className="flex h-16 shrink-0 items-center justify-between">
            <span className="font-bold text-2xl text-indigo-600 tracking-tight">Pacer</span>
            {user?.tier && (
              <span className="inline-flex items-center rounded-full bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-700 ring-1 ring-inset ring-indigo-700/10">
                {user.tier}
              </span>
            )}
          </div>
          
          {daysToRace !== null && (
             <div className="rounded-lg bg-indigo-50 p-4">
                <div className="flex items-center">
                  <Trophy className="h-5 w-5 text-indigo-600" />
                  <h3 className="ml-2 text-sm font-medium text-indigo-800">Next Race</h3>
                </div>
                <div className="mt-2 flex items-baseline gap-x-2">
                  <span className="text-3xl font-bold tracking-tight text-indigo-600">{daysToRace}</span>
                  <span className="text-sm font-medium text-indigo-800">days to go</span>
                </div>
             </div>
          )}

          <nav className="flex flex-1 flex-col">
            <ul role="list" className="flex flex-1 flex-col gap-y-7">
              <li>
                <ul role="list" className="-mx-2 space-y-1">
                  {navigation.map((item) => {
                    const isCurrent = pathname === item.href;
                    return (
                      <li key={item.name}>
                        <Link
                          href={item.href}
                          className={`
                            group flex gap-x-3 rounded-md p-2 text-sm leading-6 font-semibold
                            ${isCurrent ? 'bg-gray-50 text-indigo-600' : 'text-gray-700 hover:bg-gray-50 hover:text-indigo-600'}
                          `}
                        >
                          <item.icon
                            className={`h-6 w-6 shrink-0 ${isCurrent ? 'text-indigo-600' : 'text-gray-400 group-hover:text-indigo-600'}`}
                            aria-hidden="true"
                          />
                          {item.name}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </li>
              <li className="mt-auto">
                <button
                  onClick={handleLogout}
                  className="group -mx-2 flex w-full gap-x-3 rounded-md p-2 text-sm font-semibold leading-6 text-gray-700 hover:bg-gray-50 hover:text-indigo-600"
                >
                  <LogOut className="h-6 w-6 shrink-0 text-gray-400 group-hover:text-indigo-600" aria-hidden="true" />
                  Logout
                </button>
              </li>
            </ul>
          </nav>
        </div>
      </div>

      <div className="lg:pl-72">
        <div className="sticky top-0 z-40 flex h-16 shrink-0 items-center gap-x-4 border-b border-gray-200 bg-white px-4 shadow-sm sm:gap-x-6 sm:px-6 lg:hidden">
          <button type="button" className="-m-2.5 p-2.5 text-gray-700 lg:hidden" onClick={() => setSidebarOpen(true)}>
            <span className="sr-only">Open sidebar</span>
            <Menu className="h-6 w-6" aria-hidden="true" />
          </button>

          <div className="flex flex-1 gap-x-4 self-stretch lg:gap-x-6">
            <div className="flex flex-1 items-center font-bold text-lg text-indigo-600">Pacer</div>
            <div className="flex items-center gap-x-4 lg:gap-x-6">
               {user?.tier && (
                <span className="inline-flex items-center rounded-full bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-700 ring-1 ring-inset ring-indigo-700/10">
                  {user.tier}
                </span>
               )}
            </div>
          </div>
        </div>

        <main className="py-10">
          <div className="px-4 sm:px-6 lg:px-8">
            <ConnectBanner />
            {children}
          </div>
        </main>
      </div>
      <NotificationToaster />
    </div>
  );
}
