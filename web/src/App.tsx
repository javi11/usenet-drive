'use client'

import {
  IconButton,
  Box,
  CloseButton,
  Flex,
  Icon,
  useColorModeValue,
  Text,
  Drawer,
  DrawerContent,
  useDisclosure,
  BoxProps,
  FlexProps,
  Image,
} from '@chakra-ui/react'
import {
  FiHome,
  FiMenu,
} from 'react-icons/fi'
import {
  TbProgressDown,
  TbProgress,
  TbProgressX,
} from 'react-icons/tb'
import { IconType } from 'react-icons'
import { Link, Routes, Route } from 'react-router-dom';
import Home from './pages/Home'
import FailedJobs from './pages/FailedJobs'
import PendingJobs from './pages/PendingJobs'
import InProgressJobs from './pages/InProgressJobs'
import NotFound from './pages/NotFound'
import reactLogo from './assets/logo.png'

interface LinkItemProps {
  name: string
  icon: IconType
  path: string
}
const LinkItems: Array<LinkItemProps> = [
  { name: 'Home', icon: FiHome, path: '/' },
  { name: 'In progress jobs', icon: TbProgressDown, path: '/in-progress' },
  { name: 'Pending jobs', icon: TbProgress, path: '/pending' },
  { name: 'Failed jobs', icon: TbProgressX, path: '/failed' },
]

export default function App() {
  const { isOpen, onOpen, onClose } = useDisclosure()
  return (
    <Box minH="100vh" bg={useColorModeValue('gray.100', 'gray.900')}>
      <SidebarContent onClose={() => onClose} display={{ base: 'none', md: 'block' }} />
      <Drawer
        isOpen={isOpen}
        placement="left"
        onClose={onClose}
        returnFocusOnClose={false}
        onOverlayClick={onClose}
        size="full">
        <DrawerContent>
          <SidebarContent onClose={onClose} />
        </DrawerContent>
      </Drawer>
      {/* mobilenav */}
      <MobileNav display={{ base: 'flex', md: 'none' }} onOpen={onOpen} />
      <Box ml={{ base: 0, md: 60 }} p="4">
        <Routes>
          <Route index path="/" element={<Home />} />
          <Route path="/in-progress" element={<InProgressJobs />} />
          <Route path="/pending" element={<PendingJobs />} />
          <Route path="/failed" element={<FailedJobs />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </Box>
    </Box>
  )
}

interface SidebarProps extends BoxProps {
  onClose: () => void
}

const SidebarContent = ({ onClose, ...rest }: SidebarProps) => {
  return (
    <Box
      bg={useColorModeValue('white', 'gray.900')}
      borderRight="1px"
      borderRightColor={useColorModeValue('gray.200', 'gray.700')}
      w={{ base: 'full', md: 60 }}
      pos="fixed"
      h="full"
      {...rest}>
      <Flex h="40" alignItems="center" mx="10" justifyContent="space-between">
        <Image src={reactLogo} alt='Usenet drive' />
        <CloseButton display={{ base: 'flex', md: 'none' }} onClick={onClose} />
      </Flex>
      {LinkItems.map((link) => (
        <NavItem key={link.name} icon={link.icon} path={link.path}>
          {link.name}
        </NavItem>
      ))}
    </Box>
  )
}

interface NavItemProps extends FlexProps {
  icon: IconType
  path: string
  children: string
}

const NavItem = ({ icon, children, path, ...rest }: NavItemProps) => {
  return (
    <Link to={path} style={{ textDecoration: 'none' }}>
      <Flex
        align="center"
        p="4"
        mx="4"
        borderRadius="lg"
        role="group"
        cursor="pointer"
        _hover={{
          bg: 'cyan.400',
          color: 'white',
        }}
        {...rest}>
        {icon && (
          <Icon
            mr="4"
            fontSize="16"
            _groupHover={{
              color: 'white',
            }}
            as={icon}
          />
        )}
        {children}
      </Flex>
    </Link>
  )
}

interface MobileProps extends FlexProps {
  onOpen: () => void
}
const MobileNav = ({ onOpen, ...rest }: MobileProps) => {
  return (
    <Flex
      ml={{ base: 0, md: 60 }}
      px={{ base: 4, md: 24 }}
      height="20"
      alignItems="center"
      bg={useColorModeValue('white', 'gray.900')}
      borderBottomWidth="1px"
      borderBottomColor={useColorModeValue('gray.200', 'gray.700')}
      justifyContent="flex-start"
      {...rest}>
      <IconButton
        variant="outline"
        onClick={onOpen}
        aria-label="open menu"
        icon={<FiMenu />}
      />

      <Text fontSize="2xl" ml="8" fontFamily="monospace" fontWeight="bold">
        Logo
      </Text>
    </Flex>
  )
}
